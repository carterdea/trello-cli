package trello

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Scale-Flow/trello-cli/internal/contract"
)

func (c *Client) ListAttachments(ctx context.Context, cardID string) ([]Attachment, error) {
	var attachments []Attachment
	err := c.Get(ctx, fmt.Sprintf("/1/cards/%s/attachments", cardID), nil, &attachments)
	return attachments, err
}

func (c *Client) GetAttachment(ctx context.Context, cardID, attachmentID string) (Attachment, error) {
	var attachment Attachment
	err := c.Get(ctx, fmt.Sprintf("/1/cards/%s/attachments/%s", cardID, attachmentID), nil, &attachment)
	return attachment, err
}

func (c *Client) DownloadAttachment(ctx context.Context, cardID, attachmentID, outputPath string, force bool) (AttachmentDownloadResult, error) {
	attachment, err := c.GetAttachment(ctx, cardID, attachmentID)
	if err != nil {
		return AttachmentDownloadResult{}, err
	}
	if err := contract.ValidateURL(attachment.URL); err != nil {
		return AttachmentDownloadResult{}, err
	}

	finalPath, err := resolveAttachmentOutputPath(outputPath, attachment)
	if err != nil {
		return AttachmentDownloadResult{}, err
	}
	if err := validateAttachmentOutputFile(finalPath, force); err != nil {
		return AttachmentDownloadResult{}, err
	}

	downloadURL, needsAuth, err := c.attachmentDownloadURL(attachment)
	if err != nil {
		return AttachmentDownloadResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return AttachmentDownloadResult{}, err
	}
	if needsAuth {
		c.setTrelloAuthorization(req)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return AttachmentDownloadResult{}, contract.NewError(contract.HTTPError, fmt.Sprintf("download failed: %v", err))
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return AttachmentDownloadResult{}, mapHTTPError(resp)
	}

	file, err := openAttachmentOutputFile(finalPath, force)
	if err != nil {
		return AttachmentDownloadResult{}, err
	}
	defer file.Close()
	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return AttachmentDownloadResult{}, contract.NewError(contract.UnknownError, fmt.Sprintf("failed to write output file: %v", err))
	}

	return AttachmentDownloadResult{
		ID:       attachment.ID,
		CardID:   cardID,
		Name:     attachment.Name,
		Path:     finalPath,
		Bytes:    written,
		MimeType: attachment.MimeType,
	}, nil
}

func (c *Client) AddURLAttachment(ctx context.Context, cardID, urlStr string, name *string) (Attachment, error) {
	queryParams := map[string]string{"url": urlStr}
	if name != nil {
		queryParams["name"] = *name
	}
	var attachment Attachment
	err := c.Post(ctx, fmt.Sprintf("/1/cards/%s/attachments", cardID), queryParams, &attachment)
	return attachment, err
}

func (c *Client) AddFileAttachment(ctx context.Context, cardID, filePath string, name *string) (Attachment, error) {
	queryParams := map[string]string{}
	if name != nil {
		queryParams["name"] = *name
	}
	var attachment Attachment
	err := c.postMultipartFile(ctx, fmt.Sprintf("/1/cards/%s/attachments", cardID), filePath, queryParams, &attachment)
	return attachment, err
}

func (c *Client) DeleteAttachment(ctx context.Context, cardID, attachmentID string) error {
	return c.Delete(ctx, fmt.Sprintf("/1/cards/%s/attachments/%s", cardID, attachmentID), nil)
}

func (c *Client) attachmentDownloadURL(attachment Attachment) (string, bool, error) {
	u, err := url.Parse(attachment.URL)
	if err != nil {
		return "", false, err
	}
	needsAuth := isTrelloHost(u.Hostname()) || (attachment.IsUpload && c.isBaseHost(u.Hostname()))
	return u.String(), needsAuth, nil
}

func (c *Client) setTrelloAuthorization(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf(`OAuth oauth_consumer_key="%s", oauth_token="%s"`, c.apiKey, c.token))
}

func (c *Client) isBaseHost(host string) bool {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Hostname(), host)
}

func isTrelloHost(host string) bool {
	host = strings.ToLower(host)
	return host == "trello.com" || strings.HasSuffix(host, ".trello.com")
}

func resolveAttachmentOutputPath(outputPath string, attachment Attachment) (string, error) {
	if outputPath == "" {
		return "", contract.NewError(contract.ValidationError, "output path is required")
	}
	info, err := os.Stat(outputPath)
	if err == nil && info.IsDir() {
		name := attachmentDownloadFilename(attachment)
		if name == "" {
			return "", contract.NewError(contract.ValidationError, "could not derive attachment filename")
		}
		return filepath.Join(outputPath, name), nil
	}
	if err != nil && !os.IsNotExist(err) {
		return "", contract.NewError(contract.UnknownError, fmt.Sprintf("cannot inspect output path: %v", err))
	}
	return outputPath, nil
}

func attachmentDownloadFilename(attachment Attachment) string {
	for _, candidate := range []string{
		attachment.FileName,
		attachment.Name,
		urlPathBase(attachment.URL),
		attachment.ID,
	} {
		if name := cleanFilename(candidate); name != "" {
			return name
		}
	}
	return ""
}

func urlPathBase(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return path.Base(u.Path)
}

func cleanFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	if name == "." || name == ".." || name == "/" {
		return ""
	}
	return name
}

func validateAttachmentOutputFile(path string, force bool) error {
	if path == "" {
		return contract.NewError(contract.ValidationError, "output path is required")
	}
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return contract.NewError(contract.ValidationError, fmt.Sprintf("output path is a directory: %s", path))
		}
		if force {
			return nil
		}
		return contract.NewError(contract.Conflict, fmt.Sprintf("output file already exists: %s", path))
	}
	if !os.IsNotExist(err) {
		return contract.NewError(contract.UnknownError, fmt.Sprintf("cannot inspect output path: %v", err))
	}
	parent := filepath.Dir(path)
	if parent == "." || parent == "" {
		return nil
	}
	parentInfo, err := os.Stat(parent)
	if err != nil {
		return contract.NewError(contract.FileNotFound, fmt.Sprintf("output directory not found: %s", parent))
	}
	if !parentInfo.IsDir() {
		return contract.NewError(contract.ValidationError, fmt.Sprintf("output parent is not a directory: %s", parent))
	}
	return nil
}

func openAttachmentOutputFile(path string, force bool) (*os.File, error) {
	flags := os.O_CREATE | os.O_WRONLY
	if force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(path, flags, 0o600)
	if err == nil {
		return file, nil
	}
	if os.IsExist(err) {
		return nil, contract.NewError(contract.Conflict, fmt.Sprintf("output file already exists: %s", path))
	}
	return nil, contract.NewError(contract.UnknownError, fmt.Sprintf("cannot create output file: %v", err))
}

// postMultipartFile handles multipart/form-data file uploads.
func (c *Client) postMultipartFile(ctx context.Context, path, filePath string, params map[string]string, result any) error {
	file, err := os.Open(filePath)
	if err != nil {
		return contract.NewError(contract.FileNotFound, fmt.Sprintf("cannot open file: %s", filePath))
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return contract.NewError(contract.UnknownError, fmt.Sprintf("failed to create form file: %v", err))
	}
	if _, err := io.Copy(part, file); err != nil {
		return contract.NewError(contract.UnknownError, fmt.Sprintf("failed to read file: %v", err))
	}
	for k, v := range params {
		if err := writer.WriteField(k, v); err != nil {
			return contract.NewError(contract.UnknownError, fmt.Sprintf("failed to write form field: %v", err))
		}
	}
	if err := writer.Close(); err != nil {
		return contract.NewError(contract.UnknownError, fmt.Sprintf("failed to finalize multipart body: %v", err))
	}

	fullURL, err := c.buildURL(path, nil)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return contract.NewError(contract.HTTPError, fmt.Sprintf("upload failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return mapHTTPError(resp)
	}
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}
