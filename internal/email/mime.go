package email

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
)

func getCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".cache", "icloud", "attachments"), nil
}

func ParseMIMEMessage(rawBody []byte, mailbox string, uid uint32) (textBody, htmlBody string, attachments []Attachment, err error) {
	r, err := mail.CreateReader(bytes.NewReader(rawBody))
	if err != nil {
		if message.IsUnknownCharset(err) {
			return string(rawBody), "", nil, nil
		}
		return "", "", nil, fmt.Errorf("failed to create mail reader: %w", err)
	}
	defer r.Close()

	cacheDir, err := getCacheDir()
	if err != nil {
		return "", "", nil, err
	}
	attachmentDir := filepath.Join(cacheDir, sanitizeFilename(mailbox), fmt.Sprintf("%d", uid))

	for {
		part, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		switch h := part.Header.(type) {
		case *mail.InlineHeader:
			contentType, params, _ := h.ContentType()
			body, err := io.ReadAll(part.Body)
			if err != nil {
				continue
			}

			if strings.HasPrefix(contentType, "text/plain") {
				if textBody == "" {
					textBody = string(body)
				}
			} else if strings.HasPrefix(contentType, "text/html") {
				if htmlBody == "" {
					htmlBody = string(body)
				}
			} else if isMediaType(contentType) {
				filename := getFilenameFromParams(params, h)
				if filename == "" {
					ext := getExtensionForContentType(contentType)
					filename = fmt.Sprintf("inline_%d%s", len(attachments)+1, ext)
				}

				if err := os.MkdirAll(attachmentDir, 0755); err != nil {
					continue
				}

				safeName := sanitizeFilename(filename)
				filePath := filepath.Join(attachmentDir, safeName)
				if err := os.WriteFile(filePath, body, 0644); err != nil {
					continue
				}

				attachments = append(attachments, Attachment{
					Filename:    filename,
					ContentType: contentType,
					Size:        int64(len(body)),
					Path:        filePath,
				})
			}

		case *mail.AttachmentHeader:
			filename, err := h.Filename()
			if err != nil || filename == "" {
				filename = fmt.Sprintf("attachment_%d", len(attachments)+1)
			}
			filename = decodeFilename(filename)

			contentType, _, _ := h.ContentType()
			body, err := io.ReadAll(part.Body)
			if err != nil {
				continue
			}

			if err := os.MkdirAll(attachmentDir, 0755); err != nil {
				continue
			}

			safeName := sanitizeFilename(filename)
			filePath := filepath.Join(attachmentDir, safeName)
			if err := os.WriteFile(filePath, body, 0644); err != nil {
				continue
			}

			attachments = append(attachments, Attachment{
				Filename:    filename,
				ContentType: contentType,
				Size:        int64(len(body)),
				Path:        filePath,
			})
		}
	}

	return textBody, htmlBody, attachments, nil
}

func decodeFilename(filename string) string {
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(filename)
	if err != nil {
		return filename
	}
	return decoded
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\x00", "_",
	)
	sanitized := replacer.Replace(name)
	sanitized = strings.TrimSpace(sanitized)
	if sanitized == "" {
		sanitized = "unnamed"
	}
	return sanitized
}

func isMediaType(contentType string) bool {
	return strings.HasPrefix(contentType, "image/") ||
		strings.HasPrefix(contentType, "audio/") ||
		strings.HasPrefix(contentType, "video/") ||
		strings.HasPrefix(contentType, "application/")
}

func getFilenameFromParams(params map[string]string, h *mail.InlineHeader) string {
	if params["name"] != "" {
		return decodeFilename(params["name"])
	}

	_, dispParams, err := h.ContentDisposition()
	if err == nil && dispParams["filename"] != "" {
		return decodeFilename(dispParams["filename"])
	}

	return ""
}

func getExtensionForContentType(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "image/jpeg"):
		return ".jpg"
	case strings.HasPrefix(contentType, "image/png"):
		return ".png"
	case strings.HasPrefix(contentType, "image/gif"):
		return ".gif"
	case strings.HasPrefix(contentType, "image/webp"):
		return ".webp"
	case strings.HasPrefix(contentType, "application/pdf"):
		return ".pdf"
	default:
		return ""
	}
}
