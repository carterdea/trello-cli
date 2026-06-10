# Trello CLI Design Doc

## Problem Context

The CLI can upload, list, and delete card attachments, but cannot download attachment bytes to disk.

Current state:
- `cmd/trello/attachments.go` has list/add/delete commands only.
- `internal/trello/attachments.go` has list/add/delete client methods only.
- `Attachment` already includes `URL`, `Bytes`, `MimeType`, and `IsUpload`.
- All commands write JSON envelopes to stdout, so binary downloads should write to a file path, not stdout.

## Proposed Solution

Add:

```bash
trello attachments download --card <card-id> --attachment <attachment-id> --output <local-path-or-dir> [--force]
```

Behavior:
- Fetch attachment metadata from Trello.
- Download bytes from the returned attachment `url`.
- Write bytes to `--output`.
- Return JSON metadata with saved path and byte count.
- Refuse overwrite unless `--force`.
- Support both Trello-hosted uploaded files and external URL attachments.

## Goals and Non-Goals

### Goals

- Download Trello attachment content to local disk.
- Preserve agent-safe JSON stdout behavior.
- Support `--output` as a file path or existing directory.
- Add focused client and command tests.

### Non-Goals

- Streaming binary data to stdout.
- Bulk download all card attachments.
- Downloading Trello preview image variants.
- Interpreting, opening, converting, or rendering downloaded linked content.

## Design

Request path:

```text
CLI command
  -> auth.RequireAuth
  -> API.DownloadAttachment(cardID, attachmentID, outputPath, force)
  -> GET /1/cards/{card}/attachments/{attachment}
  -> GET attachment.url
  -> write output file
  -> JSON result
```

### Key Components

#### Command

Add `attachments download`.

Flags:
- `--card`
- `--attachment`
- `--output`
- `--force`

#### Client

Add:
- `GetAttachment(ctx, cardID, attachmentID) (Attachment, error)`
- `DownloadAttachment(ctx, cardID, attachmentID, outputPath string, force bool) (AttachmentDownloadResult, error)`

Use `Attachment.URL` as the download URL.

Download policy:
- `isUpload=true`: download Trello-hosted file from `Attachment.URL` with Trello auth in the OAuth `Authorization` header.
- `isUpload=false`: download the external URL directly, with no Trello auth header unless the URL host is Trello.
- Only allow `http` and `https` URLs.
- Save whatever bytes HTTP returns. Do not inspect or convert content.

Output path policy:
- If `--output ./brief.pdf`, write exactly there.
- If `--output ./downloads/`, derive filename from `fileName`, then `name`, then URL basename, then `<attachment-id>`.
- Refuse overwrite unless `--force`.

#### Result Type

```go
type AttachmentDownloadResult struct {
	ID       string `json:"id"`
	CardID   string `json:"cardId"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Bytes    int64  `json:"bytes"`
	MimeType string `json:"mimeType"`
}
```

### File Map

Existing files needing changes:
- `cmd/trello/attachments.go` (lines 15-131) -- add `download` command beside list/add/delete.
- `internal/trello/client.go` (lines 90-94) -- extend `API` interface with get/download attachment methods.
- `internal/trello/attachments.go` (lines 17-45) -- add metadata fetch and file download implementation.
- `internal/trello/types.go` (lines 81-90) -- add `AttachmentDownloadResult`; likely add `FileName` to `Attachment` for better directory output naming.
- `internal/contract/validation.go` (lines 87-95) -- add output path validation/refuse overwrite helper if kept outside command code.

Existing test files:
- `cmd/trello/attachments_test.go` (lines 14-136) -- add command tests for download success, missing flags, directory output, overwrite behavior.
- `cmd/trello/mock_test.go` (lines 43-46, 258-285, 569-577) -- add mock function and reset flags for download.
- `internal/trello/attachments_test.go` (lines 16-188) -- add API/client tests for metadata fetch, byte download, Trello auth on Trello-hosted URL, external URL without auth, and overwrite refusal.
- `internal/contract/validation_test.go` (lines 169-197) -- add output path validation tests if helper is added.

Docs:
- `docs/commands/attachments.md` (lines 5-24) -- document new command.
- `LLM.md` (lines 174-225) -- add command to agent-facing digest and validation rules.

New files:
- None expected.

Config/schema/migration files:
- None.

## Alternatives Considered

| Alternative | Pros | Cons | Why Not Chosen |
|-------------|------|------|----------------|
| Print bytes to stdout | Unix-friendly | Breaks JSON contract | CLI promises JSON stdout |
| Only expose attachment URL | Minimal | Does not solve download | User needs actual file download |
| Reject `isUpload=false` URL attachments | Conservative | Friction on trusted boards | User controls boards and wants less restriction |
| Bulk download now | More powerful | More edge cases | Keep first slice small |

## Open Questions

- [x] Should derived filenames be sanitized beyond path separators and empty names?
  - Answer: Keep sanitization small: use `filepath.Base` to drop path separators, reject empty or `.` names, then fall back through `fileName`, `name`, URL basename, and attachment ID.

## Implementation Plan

### Phase 1: Foundation

- [x] Add `AttachmentDownloadResult` and `Attachment.FileName` -- `internal/trello/types.go` (lines 81-90)
  - QA: N/A
- [x] Extend `trello.API` with `GetAttachment` and `DownloadAttachment` -- `internal/trello/client.go` (lines 90-94)
  - QA: N/A; package compile returns green after the concrete client methods in Phase 2.
- [x] Add output path validation/refuse-overwrite helper -- `internal/contract/validation.go` (lines 87-95), `internal/contract/validation_test.go` (lines 169-197)
  - QA: N/A

### Phase 2: Core Implementation

- [x] Implement `GetAttachment` using `GET /1/cards/{id}/attachments/{idAttachment}` -- `internal/trello/attachments.go` (lines 17-45)
  - QA: Run `trello attachments download --card <card-id> --attachment <attachment-id> --output ./downloaded-file`; expect metadata lookup before download.
- [x] Implement `DownloadAttachment` to fetch metadata, resolve output path, download `Attachment.URL`, and write bytes -- `internal/trello/attachments.go` (lines 17-45)
  - [ ] QA: Run `trello attachments download --card <card-id> --attachment <attachment-id> --output ./downloaded-file`; expect the file to exist and JSON to report `path` and `bytes`.
- [x] Add authenticated Trello-hosted URL handling and direct external URL handling -- `internal/trello/attachments.go` (lines 17-45)
  - [ ] QA: Download one uploaded Trello file and one URL attachment; expect both files to save without exposing binary content on stdout.
- [x] Add `attachments download` command and flags -- `cmd/trello/attachments.go` (lines 93-131)
  - [x] QA: Run `trello attachments download --card <card-id> --attachment <attachment-id> --output ./downloaded-file`; expect JSON output and no binary stdout.
    > PASS: External URL attachment download returned a JSON envelope with `path` and `bytes` and wrote content to disk without binary stdout. Artifact: `qa/attachment-download-qa-20260609204910/download-url-fresh.json`.
- [x] Update command mock and flag reset -- `cmd/trello/mock_test.go` (lines 43-46, 258-285, 569-577)
  - QA: N/A

### Phase 3: Polish & Testing

- [x] Add client tests for metadata fetch, byte write, directory output naming, auth behavior, external URL behavior, and overwrite refusal -- `internal/trello/attachments_test.go` (lines 16-188)
  - QA: N/A
- [x] Add command tests for success and validation errors -- `cmd/trello/attachments_test.go` (lines 14-136)
  - QA: N/A
- [x] Update command docs -- `docs/commands/attachments.md` (lines 5-24)
  - QA: Read `docs/commands/attachments.md`; expect `attachments download` usage, validation, and example.
- [x] Update LLM digest -- `LLM.md` (lines 174-225)
  - QA: Read `LLM.md`; expect `attachments download` syntax and overwrite validation rule.
- [x] Run focused checks: `go test ./cmd/trello ./internal/trello ./internal/contract`
  - QA: N/A

## Appendix

Context7 docs used:
- Trello REST API Cards attachments: https://developer.atlassian.com/cloud/trello/rest/api-group-cards
- Trello attachment object definition: https://developer.atlassian.com/cloud/trello/guides/rest-api/object-definitions
