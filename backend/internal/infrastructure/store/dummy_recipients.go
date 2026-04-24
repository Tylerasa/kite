package store

import (
	"context"

	"github.com/kite/internal/domain/models"
)

// seedRecipients maps "bankCode:accountNumber" to the registered account name.
// These are the "golden" test accounts that always resolve to a specific name.
var seedRecipients = map[string]string{
	// NGN — ACCESS BANK (000014)
	"000014:0123456789": "JOHN ADEWALE DOE",
	"000014:0000000001": "MARY CHIOMA OKAFOR",
	"000014:1234567890": "CHUKWUEMEKA OBI NWOSU",
	// NGN — GTBANK PLC (000013)
	"000013:9876543210": "IBRAHIM MUSA ALIYU",
	"000013:0000000002": "GRACE EMEKA NWOSU",
	"000013:5000000001": "OLUWAFEMI ADEYEMI",
	// NGN — ZENITH BANK (000015)
	"000015:5555555555": "FUNMI ADESOLA BELLO",
	"000015:0000000003": "SAMUEL TAIWO ADELEKE",
	// NGN — KUDA MICROFINANCE BANK (090267)
	"090267:1111111111": "TUNDE BABATUNDE SULE",
	"090267:2222222222": "AISHA MUHAMMED YUSUF",
	// NGN — FIRST BANK OF NIGERIA (000016)
	"000016:3013012345": "NGOZI AMAKA EZE",
	// NGN — UNITED BANK FOR AFRICA (000004)
	"000004:2000000001": "BIODUN OLAWALE AFOLABI",
	// NGN — OPAY (100004)
	"100004:0810000001": "FATIMA BELLO GARBA",
	// NGN — MONIEPOINT (090405)
	"090405:7000000001": "SUNDAY IFEANYI OKOYE",

	// KES — Kenya Commercial Bank (0001)
	"0001:0000000001": "PETER NJOROGE KAMAU",
	"0001:0000000002": "AMINA HASSAN WANJIKU",
	"0001:1234567890": "JAMES MWANGI KARIUKI",
	// KES — Equity Bank (0068)
	"0068:0000000001": "DAVID KIPCHOGE MUTAI",
	"0068:9876543210": "GRACE WANGARI NJERI",
	// KES — Cooperative Bank of Kenya (0011)
	"0011:5000000001": "JOSEPH OTIENO ODHIAMBO",
	// KES — ABSA Bank Kenya (0003)
	"0003:3000000001": "LUCY ADHIAMBO AUMA",
	// KES — SAFARICOM M-PESA (SAFKEN) — account number is mobile number
	"SAFKEN:0712345678": "WANJIRU MWANGI",
	"SAFKEN:0700000001": "KARIUKI JOHN GITAU",
	"SAFKEN:0722000001": "BEATRICE AKINYI OTIENO",
	// KES — AIRTEL (AIRKEN)
	"AIRKEN:0733000001": "HASSAN OMAR ABDI",
}

// DummyRecipientProvider implements out.RecipientInquiryProvider without a real payment network.
// Seeded accounts return known names; any other account number gets a deterministic fallback
// so that any account number the user enters during testing will "resolve" successfully.
type DummyRecipientProvider struct{}

func (p *DummyRecipientProvider) Resolve(_ context.Context, _ models.Currency, bankCode, accountNumber string) (string, error) {
	key := bankCode + ":" + accountNumber
	if name, ok := seedRecipients[key]; ok {
		return name, nil
	}
	// Deterministic fallback: last 4 digits of account number make it feel real.
	suffix := accountNumber
	if len(suffix) > 4 {
		suffix = suffix[len(suffix)-4:]
	}
	return "ACCOUNT HOLDER " + suffix, nil
}
