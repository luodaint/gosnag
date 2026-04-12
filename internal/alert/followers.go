package alert

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NotifyFollowers sends email notifications to all followers of an issue.
// action describes what happened (e.g., "Ticket created", "Status changed to in_progress").
// excludeUserID is the user who made the change (don't notify them about their own action).
func (s *Service) NotifyFollowers(issueID uuid.UUID, projectID uuid.UUID, issueTitle, action string, excludeUserID *uuid.UUID) {
	if s.cfg.SMTPHost == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	followers, err := s.queries.ListIssueFollowers(ctx, issueID)
	if err != nil || len(followers) == 0 {
		return
	}

	project, err := s.queries.GetProject(ctx, projectID)
	if err != nil {
		return
	}

	var recipients []string
	for _, f := range followers {
		if excludeUserID != nil && f.ID == *excludeUserID {
			continue
		}
		if f.Email != "" {
			recipients = append(recipients, f.Email)
		}
	}

	if len(recipients) == 0 {
		return
	}

	subject := fmt.Sprintf("[GoSnag] %s — %s", action, truncate(issueTitle, 80))
	issueURL := fmt.Sprintf("%s/projects/%s/issues/%s", s.cfg.BaseURL, projectID.String(), issueID.String())
	body := fmt.Sprintf(
		"Project: %s\nIssue: %s\nAction: %s\n\nView: %s",
		project.Name,
		issueTitle,
		action,
		issueURL,
	)

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.cfg.SMTPFrom,
		strings.Join(recipients, ", "),
		subject,
		body,
	)

	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	var auth smtp.Auth
	if s.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, s.cfg.SMTPFrom, recipients, []byte(msg)); err != nil {
		slog.Error("failed to send follower notification", "error", err, "recipients", recipients)
	} else {
		slog.Debug("follower notification sent", "issue_id", issueID, "recipients", len(recipients), "action", action)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
