package controllers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kite/internal/domain/models"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/infrastructure/rest/apiutil"
	"github.com/kite/internal/infrastructure/rest/middleware"
)

type InstitutionController struct {
	uc     in.InstitutionUseCase
	logger *slog.Logger
}

func NewInstitutionController(uc in.InstitutionUseCase, logger *slog.Logger) *InstitutionController {
	return &InstitutionController{uc: uc, logger: logger}
}

type institutionResponse struct {
	Type     string `json:"type"`
	BankCode string `json:"bank_code"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Logo     string `json:"logo,omitempty"`
}

type inquiryRequest struct {
	Currency      string `json:"currency" binding:"required"`
	BankCode      string `json:"bank_code" binding:"required"`
	AccountNumber string `json:"account_number" binding:"required"`
}

type inquiryResponse struct {
	AccountName     string `json:"account_name"`
	AccountNumber   string `json:"account_number"`
	BankCode        string `json:"bank_code"`
	BankName        string `json:"bank_name"`
	InstitutionType string `json:"institution_type"`
}

// List godoc
// @Summary      List institutions for a currency
// @Description  Returns all banks and mobile money providers available for payouts in the given currency
// @Tags         institutions
// @Produce      json
// @Param        currency  query  string  true  "Currency code (NGN, KES)"
// @Success      200  {array}   institutionResponse
// @Failure      400  {object}  apiutil.ErrorResponse
// @Security     BearerAuth
// @Router       /institutions [get]
func (ctrl *InstitutionController) List(c *gin.Context) {
	currency := models.Currency(c.Query("currency"))
	list, err := ctrl.uc.ListInstitutions(c.Request.Context(), currency)
	if err != nil {
		apiutil.RespondError(c, ctrl.logger, err)
		return
	}
	resp := make([]institutionResponse, len(list))
	for i, inst := range list {
		resp[i] = institutionResponse{
			Type:     inst.Type,
			BankCode: inst.BankCode,
			Name:     inst.Name,
			Currency: string(inst.Currency),
			Logo:     inst.Logo,
		}
	}
	c.JSON(http.StatusOK, resp)
}

// Inquiry godoc
// @Summary      Resolve recipient account name
// @Description  Performs a name enquiry: given an account number and bank code, returns the registered account holder name
// @Tags         payouts
// @Accept       json
// @Produce      json
// @Param        body  body      inquiryRequest   true  "Inquiry request"
// @Success      200   {object}  inquiryResponse
// @Failure      400   {object}  apiutil.ErrorResponse
// @Security     BearerAuth
// @Router       /payouts/inquiry [post]
func (ctrl *InstitutionController) Inquiry(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req inquiryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code: "validation_error", Message: apiutil.BindingError(err),
		}})
		return
	}

	result, err := ctrl.uc.ResolveRecipient(c.Request.Context(), in.InquiryCommand{
		UserID:        userID,
		Currency:      models.Currency(req.Currency),
		BankCode:      req.BankCode,
		AccountNumber: req.AccountNumber,
	})
	if err != nil {
		apiutil.RespondError(c, ctrl.logger, err)
		return
	}

	c.JSON(http.StatusOK, inquiryResponse{
		AccountName:     result.AccountName,
		AccountNumber:   result.AccountNumber,
		BankCode:        result.BankCode,
		BankName:        result.BankName,
		InstitutionType: result.InstitutionType,
	})
}
