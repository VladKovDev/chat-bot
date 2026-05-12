package dialogreset

import (
	"context"
	"strings"

	domaindialogreset "github.com/VladKovDev/chat-bot/internal/domain/dialogreset"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

type Service struct {
	repo   domaindialogreset.Repository
	logger logger.Logger
}

func NewService(repo domaindialogreset.Repository, logger logger.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) ResetSession(
	ctx context.Context,
	req domaindialogreset.Request,
) (domaindialogreset.Summary, error) {
	req.Actor = strings.TrimSpace(req.Actor)
	if req.Actor == "" {
		req.Actor = "system"
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" {
		req.Reason = "manual_reset"
	}

	summary, err := s.repo.ResetSession(ctx, req)
	if err != nil {
		return domaindialogreset.Summary{}, err
	}

	s.logger.Info("dialog session reset",
		s.logger.String("session_id", req.SessionID.String()),
		s.logger.String("actor", req.Actor),
		s.logger.String("reason", req.Reason),
		s.logger.Bool("existed", summary.Existed),
		s.logger.Any("deleted", summary.Deleted),
	)
	return summary, nil
}
