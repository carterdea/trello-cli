package main

import (
	"github.com/Scale-Flow/trello-cli/internal/auth"
	"github.com/Scale-Flow/trello-cli/internal/contract"
	"github.com/Scale-Flow/trello-cli/internal/trello"
	"github.com/spf13/cobra"
)

var attachmentsCmd = &cobra.Command{
	Use:   "attachments",
	Short: "Manage card attachments",
}

var attachmentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List attachments on a card",
	RunE: func(cmd *cobra.Command, args []string) error {
		cardID, _ := cmd.Flags().GetString("card")
		if err := contract.RequireFlag("card", cardID); err != nil {
			return err
		}
		creds, err := auth.RequireAuth(getCredStore(), "default")
		if err != nil {
			return err
		}
		attachments, err := getAPIClient(creds).ListAttachments(cmd.Context(), cardID)
		if err != nil {
			return err
		}
		return output(cmd.OutOrStdout(), attachments)
	},
}

var attachmentsAddFileCmd = &cobra.Command{
	Use:   "add-file",
	Short: "Attach a local file to a card",
	RunE: func(cmd *cobra.Command, args []string) error {
		cardID, _ := cmd.Flags().GetString("card")
		path, _ := cmd.Flags().GetString("path")
		name, _ := cmd.Flags().GetString("name")
		if err := contract.RequireFlag("card", cardID); err != nil {
			return err
		}
		if err := contract.ValidateFilePath(path); err != nil {
			return err
		}
		var namePtr *string
		if name != "" {
			namePtr = &name
		}
		creds, err := auth.RequireAuth(getCredStore(), "default")
		if err != nil {
			return err
		}
		attachment, err := getAPIClient(creds).AddFileAttachment(cmd.Context(), cardID, path, namePtr)
		if err != nil {
			return err
		}
		return output(cmd.OutOrStdout(), attachment)
	},
}

var attachmentsAddURLCmd = &cobra.Command{
	Use:   "add-url",
	Short: "Attach a URL to a card",
	RunE: func(cmd *cobra.Command, args []string) error {
		cardID, _ := cmd.Flags().GetString("card")
		urlStr, _ := cmd.Flags().GetString("url")
		name, _ := cmd.Flags().GetString("name")
		if err := contract.RequireFlag("card", cardID); err != nil {
			return err
		}
		if err := contract.ValidateURL(urlStr); err != nil {
			return err
		}
		var namePtr *string
		if name != "" {
			namePtr = &name
		}
		creds, err := auth.RequireAuth(getCredStore(), "default")
		if err != nil {
			return err
		}
		attachment, err := getAPIClient(creds).AddURLAttachment(cmd.Context(), cardID, urlStr, namePtr)
		if err != nil {
			return err
		}
		return output(cmd.OutOrStdout(), attachment)
	},
}

var attachmentsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an attachment from a card",
	RunE: func(cmd *cobra.Command, args []string) error {
		cardID, _ := cmd.Flags().GetString("card")
		attachmentID, _ := cmd.Flags().GetString("attachment")
		if err := contract.RequireFlag("card", cardID); err != nil {
			return err
		}
		if err := contract.RequireFlag("attachment", attachmentID); err != nil {
			return err
		}
		creds, err := auth.RequireAuth(getCredStore(), "default")
		if err != nil {
			return err
		}
		if err := getAPIClient(creds).DeleteAttachment(cmd.Context(), cardID, attachmentID); err != nil {
			return err
		}
		return output(cmd.OutOrStdout(), trello.DeleteResult{Deleted: true, ID: attachmentID})
	},
}

var attachmentsDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download an attachment to a local file",
	RunE: func(cmd *cobra.Command, args []string) error {
		cardID, _ := cmd.Flags().GetString("card")
		attachmentID, _ := cmd.Flags().GetString("attachment")
		outputPath, _ := cmd.Flags().GetString("output")
		force, _ := cmd.Flags().GetBool("force")
		if err := contract.RequireFlag("card", cardID); err != nil {
			return err
		}
		if err := contract.RequireFlag("attachment", attachmentID); err != nil {
			return err
		}
		if err := contract.RequireFlag("output", outputPath); err != nil {
			return err
		}
		creds, err := auth.RequireAuth(getCredStore(), "default")
		if err != nil {
			return err
		}
		result, err := getAPIClient(creds).DownloadAttachment(cmd.Context(), cardID, attachmentID, outputPath, force)
		if err != nil {
			return err
		}
		return output(cmd.OutOrStdout(), result)
	},
}

func init() {
	attachmentsListCmd.Flags().String("card", "", "Card ID")
	attachmentsAddFileCmd.Flags().String("card", "", "Card ID")
	attachmentsAddFileCmd.Flags().String("path", "", "File path")
	attachmentsAddFileCmd.Flags().String("name", "", "Attachment name")
	attachmentsAddURLCmd.Flags().String("card", "", "Card ID")
	attachmentsAddURLCmd.Flags().String("url", "", "Attachment URL")
	attachmentsAddURLCmd.Flags().String("name", "", "Attachment name")
	attachmentsDeleteCmd.Flags().String("card", "", "Card ID")
	attachmentsDeleteCmd.Flags().String("attachment", "", "Attachment ID")
	attachmentsDownloadCmd.Flags().String("card", "", "Card ID")
	attachmentsDownloadCmd.Flags().String("attachment", "", "Attachment ID")
	attachmentsDownloadCmd.Flags().String("output", "", "Output file or directory")
	attachmentsDownloadCmd.Flags().Bool("force", false, "Overwrite output file if it exists")

	attachmentsCmd.AddCommand(attachmentsListCmd)
	attachmentsCmd.AddCommand(attachmentsAddFileCmd)
	attachmentsCmd.AddCommand(attachmentsAddURLCmd)
	attachmentsCmd.AddCommand(attachmentsDeleteCmd)
	attachmentsCmd.AddCommand(attachmentsDownloadCmd)
	rootCmd.AddCommand(attachmentsCmd)
}
