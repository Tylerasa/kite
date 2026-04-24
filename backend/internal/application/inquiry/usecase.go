package inquiry

import (
	"context"

	"github.com/kite/internal/domain/exceptions"
	"github.com/kite/internal/domain/models"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/domain/ports/out"
)

type UseCase struct {
	institutions out.InstitutionStore
	resolver     out.RecipientInquiryProvider
}

func NewUseCase(institutions out.InstitutionStore, resolver out.RecipientInquiryProvider) *UseCase {
	return &UseCase{institutions: institutions, resolver: resolver}
}

func (uc *UseCase) ListInstitutions(ctx context.Context, currency models.Currency) ([]*models.Institution, error) {
	if !currency.Valid() {
		return nil, exceptions.ErrInvalidCurrency
	}
	return uc.institutions.ListByCurrency(currency)
}

func (uc *UseCase) ResolveRecipient(ctx context.Context, cmd in.InquiryCommand) (*in.InquiryResult, error) {
	if !cmd.Currency.Valid() {
		return nil, exceptions.ErrInvalidCurrency
	}
	if cmd.BankCode == "" {
		return nil, exceptions.ErrInvalidBankCode
	}
	if cmd.AccountNumber == "" {
		return nil, exceptions.ErrInternal.WithDetails(map[string]interface{}{
			"field": "account_number", "reason": "required",
		})
	}

	inst, err := uc.institutions.GetByCode(cmd.Currency, cmd.BankCode)
	if err != nil {
		return nil, err
	}

	accountName, err := uc.resolver.Resolve(ctx, cmd.Currency, cmd.BankCode, cmd.AccountNumber)
	if err != nil {
		return nil, err
	}

	bankName := cmd.BankCode
	institutionType := "BANK_TRANSFER"
	if inst != nil {
		bankName = inst.Name
		institutionType = inst.Type
	}

	return &in.InquiryResult{
		AccountName:     accountName,
		AccountNumber:   cmd.AccountNumber,
		BankCode:        cmd.BankCode,
		BankName:        bankName,
		InstitutionType: institutionType,
	}, nil
}
