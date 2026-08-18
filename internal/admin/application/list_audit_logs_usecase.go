package application

import "context"

type ListAuditLogsUseCase struct {
	repo AdminRepository
}

func NewListAuditLogsUseCase(repo AdminRepository) *ListAuditLogsUseCase {
	return &ListAuditLogsUseCase{repo: repo}
}

func (uc *ListAuditLogsUseCase) Execute(ctx context.Context, limit, offset int) ([]AuditLogEntry, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	return uc.repo.ListAuditLogs(ctx, limit, offset)
}
