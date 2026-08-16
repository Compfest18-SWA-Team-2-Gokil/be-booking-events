package application

import "context"

type ListAuditLogsUseCase struct {
	repo AdminRepository
}

func NewListAuditLogsUseCase(repo AdminRepository) *ListAuditLogsUseCase {
	return &ListAuditLogsUseCase{repo: repo}
}

func (uc *ListAuditLogsUseCase) Execute(ctx context.Context, limit int) ([]AuditLogEntry, error) {
	return uc.repo.ListAuditLogs(ctx, limit)
}
