# `trello attachments`

Manage card attachments.

## Subcommands

- `trello attachments list --card <card-id>`
- `trello attachments add-file --card <card-id> --path <local-path> [--name <display-name>]`
- `trello attachments add-url --card <card-id> --url <http-or-https-url> [--name <display-name>]`
- `trello attachments download --card <card-id> --attachment <attachment-id> --output <local-path-or-dir> [--force]`
- `trello attachments delete --card <card-id> --attachment <attachment-id>`

## Validation

- `add-file` requires a local file that exists
- `add-url` requires a valid `http` or `https` URL
- `download` requires the card ID, attachment ID, and output path
- `download` writes to a file path, or derives a filename when `--output` is an existing directory
- `download` refuses to overwrite an existing file unless `--force` is provided
- `delete` requires both the card ID and attachment ID

## Examples

```bash
trello attachments list --card <card-id>
trello attachments add-file --card <card-id> --path ./brief.pdf --name "Project brief"
trello attachments add-url --card <card-id> --url https://example.com/spec --name "Spec"
trello attachments download --card <card-id> --attachment <attachment-id> --output ./downloads/
trello attachments delete --card <card-id> --attachment <attachment-id>
```
