package usecase

import (
	"encoding/xml"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain/error"
)

const (
	ndflKND = "1151020"
	ndflKBK = "18210102010011000110"
)

var digitsOnly = regexp.MustCompile(`\D`)

func buildNDFL3XML(req domain.NDFLRenderRequest, programVersion, formatVersion string) ([]byte, error) {
	now := time.Now().UTC()

	header := req.Header
	taxYear := int(header.TaxYear)
	if taxYear <= 0 {
		return nil, apperr.Internal("invalid ndfl header tax_year", nil, map[string]string{
			"field": "header.tax_year",
		})
	}

	inn, err := normalizeINN(header.INN)
	if err != nil {
		return nil, err
	}
	oktmo, err := normalizeOKTMO(header.OKTMO)
	if err != nil {
		return nil, err
	}
	taxOfficeCode, err := normalizeTaxOfficeCode(header.TaxOfficeCode)
	if err != nil {
		return nil, err
	}
	period, err := requiredTrimmed("header.tax_period_code", header.TaxPeriodCode)
	if err != nil {
		return nil, err
	}
	correctionNumber, err := normalizeCorrectionNumber(header.CorrectionNumber)
	if err != nil {
		return nil, err
	}
	lastName, err := requiredTrimmed("header.last_name", header.LastName)
	if err != nil {
		return nil, err
	}
	firstName, err := requiredTrimmed("header.first_name", header.FirstName)
	if err != nil {
		return nil, err
	}
	docPresenter, err := taxPayerTypeToDocPresenter(header.TaxPayerType)
	if err != nil {
		return nil, err
	}
	status, err := taxResidencyToStatus(header.TaxResidency)
	if err != nil {
		return nil, err
	}

	fileID, err := requiredTrimmed("report_id", req.ReportID)
	if err != nil {
		return nil, err
	}

	section1KBK, err := normalizeKBK(req.Section1.KBK)
	if err != nil {
		return nil, err
	}
	section1OKTMO, err := normalizeOKTMO(req.Section1.OKTMO)
	if err != nil {
		return nil, err
	}
	if section1OKTMO != oktmo {
		return nil, apperr.Internal("header.oktmo and section1.oktmo mismatch", nil, map[string]string{
			"field": "section1.oktmo",
		})
	}
	section1TaxToPay, err := normalizeInt(req.Section1.TaxToPay, "section1.tax_to_pay")
	if err != nil {
		return nil, err
	}
	section1TaxToRefund, err := normalizeInt(req.Section1.TaxToRefund, "section1.tax_to_refund")
	if err != nil {
		return nil, err
	}

	incomeGroupCode, err := requiredTrimmed("section2.income_group_code", req.Section2.IncomeGroupCode)
	if err != nil {
		return nil, err
	}
	totalIncome, err := normalizeDecimal2(req.Section2.TotalIncome, "section2.total_income")
	if err != nil {
		return nil, err
	}
	nonTaxableIncome, err := normalizeDecimal2(req.Section2.NonTaxableIncome, "section2.non_taxable_income")
	if err != nil {
		return nil, err
	}
	taxableIncome, err := normalizeDecimal2(req.Section2.TaxableIncome, "section2.taxable_income")
	if err != nil {
		return nil, err
	}
	deductions, err := normalizeDecimal2(req.Section2.Deductions, "section2.deductions")
	if err != nil {
		return nil, err
	}
	recognizedExpenses, err := normalizeDecimal2(req.Section2.RecognizedExpenses, "section2.recognized_expenses")
	if err != nil {
		return nil, err
	}
	taxBase, err := normalizeDecimal2(req.Section2.TaxBase, "section2.tax_base")
	if err != nil {
		return nil, err
	}
	calculatedTax, err := normalizeInt(req.Section2.CalculatedTax, "section2.calculated_tax")
	if err != nil {
		return nil, err
	}
	withheldAtSource, err := normalizeInt(req.Section2.WithheldAtSource, "section2.withheld_at_source")
	if err != nil {
		return nil, err
	}
	materialBenefitTax, err := normalizeInt(req.Section2.MaterialBenefitTax, "section2.material_benefit_tax")
	if err != nil {
		return nil, err
	}
	tradingFeeCredit, err := normalizeInt(req.Section2.TradingFeeCredit, "section2.trading_fee_credit")
	if err != nil {
		return nil, err
	}
	fixedAdvanceCredit, err := normalizeInt(req.Section2.FixedAdvanceCredit, "section2.fixed_advance_credit")
	if err != nil {
		return nil, err
	}
	foreignTaxCredit, err := normalizeInt(req.Section2.ForeignTaxCredit, "section2.foreign_tax_credit")
	if err != nil {
		return nil, err
	}
	patentTaxCredit, err := normalizeInt(req.Section2.PatentTaxCredit, "section2.patent_tax_credit")
	if err != nil {
		return nil, err
	}
	section2TaxToPay, err := normalizeInt(req.Section2.TaxToPay, "section2.tax_to_pay")
	if err != nil {
		return nil, err
	}
	section2TaxToRefund, err := normalizeInt(req.Section2.TaxToRefund, "section2.tax_to_refund")
	if err != nil {
		return nil, err
	}
	simplifiedDeductionRefund, err := normalizeInt(req.Section2.SimplifiedDeductionRefund, "section2.simplified_deduction_refund")
	if err != nil {
		return nil, err
	}

	appendix2, err := buildAppendix2Lines(req.Appendix2)
	if err != nil {
		return nil, err
	}

	appendix6Deduction, err := normalizeDecimal2(req.Appendix6.OtherPropertyDeduction, "appendix6.other_property_deduction")
	if err != nil {
		return nil, err
	}
	appendix6AcquisitionExp, err := normalizeDecimal2(req.Appendix6.OtherPropertyAcquisitionExp, "appendix6.other_property_acquisition_exp")
	if err != nil {
		return nil, err
	}
	appendix6TotalDeduction, err := normalizeDecimal2(req.Appendix6.TotalPropertyDeduction, "appendix6.total_property_deduction")
	if err != nil {
		return nil, err
	}

	document := ndflDocumentXML{
		KND:              ndflKND,
		DateDoc:          now.Format("02.01.2006"),
		Period:           period,
		ReportYear:       taxYear,
		TaxOfficeCode:    taxOfficeCode,
		CorrectionNumber: correctionNumber,
		Taxpayer: ndflTaxpayerXML{
			Person: ndflPersonXML{
				FIO: ndflFIOXML{
					LastName:   lastName,
					FirstName:  firstName,
					MiddleName: strings.TrimSpace(header.MiddleName),
				},
				INN:          inn,
				DocPresenter: docPresenter,
				Status:       status,
				Phone:        strings.TrimSpace(header.Phone),
			},
		},
		Signer: ndflSignerXML{
			SignerType: "1",
		},
		Body: ndflBodyXML{
			TotalTax: ndflTotalTaxXML{
				Excl227: []ndflTotalTaxLineXML{
					{
						KBK:         section1KBK,
						OKTMO:       section1OKTMO,
						TaxToPay:    section1TaxToPay,
						TaxToRefund: section1TaxToRefund,
					},
				},
			},
			TaxBase: []ndflTaxBaseXML{
				{
					IncomeGroupCode: incomeGroupCode,
					CalcTaxBase: ndflCalcTaxBaseXML{
						TotalIncome:        totalIncome,
						NonTaxableIncome:   nonTaxableIncome,
						TaxableIncome:      taxableIncome,
						Deductions:         deductions,
						RecognizedExpenses: recognizedExpenses,
						TaxBase:            taxBase,
					},
					CalcTaxToPay: ndflCalcTaxToPayXML{
						CalculatedTax:             calculatedTax,
						WithheldAtSource:          withheldAtSource,
						MaterialBenefitTax:        materialBenefitTax,
						TradingFeeCredit:          tradingFeeCredit,
						FixedAdvanceCredit:        fixedAdvanceCredit,
						ForeignTaxCredit:          foreignTaxCredit,
						PatentTaxCredit:           patentTaxCredit,
						TaxToPay:                  section2TaxToPay,
						TaxToRefund:               section2TaxToRefund,
						SimplifiedDeductionRefund: simplifiedDeductionRefund,
					},
				},
			},
			Appendix6: &ndflAppendix6XML{
				TotalPropertyDeduction: appendix6TotalDeduction,
				OtherPropertySale: ndflSumPr6XML{
					OtherPropertyDeduction:      appendix6Deduction,
					OtherPropertyAcquisitionExp: appendix6AcquisitionExp,
				},
			},
		},
	}

	if len(appendix2) > 0 {
		document.Body.Appendix2 = &ndflAppendix2XML{Lines: appendix2}
	}

	root := ndflFileXML{
		FileID:         fileID,
		ProgramVersion: fallbackNonEmpty(programVersion, "report-svc"),
		FormatVersion:  fallbackNonEmpty(formatVersion, "5.20"),
		Document:       document,
	}

	out, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, apperr.RenderingFailed("marshal ndfl xml failed", err, nil)
	}

	final := make([]byte, 0, len(xml.Header)+len(out))
	final = append(final, []byte(xml.Header)...)
	final = append(final, out...)
	final = append(final, '\n')
	return final, nil
}

func buildAppendix2Lines(lines []domain.NDFLAppendix2Line) ([]ndflAppendix2LineXML, error) {
	if len(lines) == 0 {
		return nil, nil
	}

	out := make([]ndflAppendix2LineXML, 0, len(lines))
	for i, line := range lines {
		sourceCountryCode, err := normalizeCountryCode(line.SourceCountryCode, "appendix2.source_country_code", i)
		if err != nil {
			return nil, err
		}
		paymentCountryCode, err := normalizeCountryCode(line.PaymentCountryCode, "appendix2.payment_country_code", i)
		if err != nil {
			return nil, err
		}
		sourceName, err := requiredTrimmed(fmt.Sprintf("appendix2[%d].source_name", i), line.SourceName)
		if err != nil {
			return nil, err
		}
		currencyCode, err := normalizeCurrencyCode(line.CurrencyCode, i)
		if err != nil {
			return nil, err
		}
		incomeTypeCode, err := normalizeIncomeTypeCode(line.IncomeTypeCode, i)
		if err != nil {
			return nil, err
		}
		if line.IncomeDate.IsZero() {
			return nil, apperr.Internal("appendix2 income_date is required", nil, map[string]string{
				"field": "appendix2.income_date",
				"index": strconv.Itoa(i),
			})
		}
		fxRate, err := normalizeRate(line.FXRate, fmt.Sprintf("appendix2[%d].fx_rate", i))
		if err != nil {
			return nil, err
		}
		incomeForeign, err := normalizeDecimal2(line.IncomeForeign, fmt.Sprintf("appendix2[%d].income_foreign", i))
		if err != nil {
			return nil, err
		}
		incomeRub, err := normalizeDecimal2(line.IncomeRub, fmt.Sprintf("appendix2[%d].income_rub", i))
		if err != nil {
			return nil, err
		}
		out = append(out, ndflAppendix2LineXML{
			SourceCountryCode:  sourceCountryCode,
			PaymentCountryCode: paymentCountryCode,
			SourceName:         sourceName,
			CurrencyCode:       currencyCode,
			IncomeTypeCode:     incomeTypeCode,
			IncomeDate:         line.IncomeDate.UTC().Format("02.01.2006"),
			FXRate:             fxRate,
			IncomeForeign:      incomeForeign,
			IncomeRub:          incomeRub,
		})
	}
	return out, nil
}

func fallbackNonEmpty(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func requiredTrimmed(field, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", apperr.Internal("required ndfl field is empty", nil, map[string]string{
			"field": field,
		})
	}
	return trimmed, nil
}

func normalizeINN(value string) (string, error) {
	digits := normalizeDigits(value)
	if len(digits) != 12 {
		return "", apperr.Internal("invalid INN value for ndfl", nil, map[string]string{
			"field": "header.inn",
		})
	}
	if len(strings.TrimSpace(value)) != len(digits) {
		return "", apperr.Internal("invalid INN value for ndfl", nil, map[string]string{
			"field": "header.inn",
		})
	}
	if !isValidIndividualINN(digits) {
		return "", apperr.Internal("invalid INN value for ndfl", nil, map[string]string{
			"field": "header.inn",
		})
	}
	return digits, nil
}

func normalizeTaxOfficeCode(value string) (string, error) {
	digits := normalizeDigits(value)
	if len(digits) != 4 || len(strings.TrimSpace(value)) != len(digits) {
		return "", apperr.Internal("invalid tax office code for ndfl", nil, map[string]string{
			"field": "header.tax_office_code",
		})
	}
	return digits, nil
}

func normalizeOKTMO(value string) (string, error) {
	digits := normalizeDigits(value)
	if (len(digits) != 8 && len(digits) != 11) || len(strings.TrimSpace(value)) != len(digits) {
		return "", apperr.Internal("invalid OKTMO value for ndfl", nil, map[string]string{
			"field": "header.oktmo",
		})
	}
	return digits, nil
}

func normalizeDigitsLen(value string, length int, field string) (string, error) {
	digits := normalizeDigits(value)
	if len(digits) != length || len(strings.TrimSpace(value)) != len(digits) {
		return "", apperr.Internal("invalid digits length for ndfl field", nil, map[string]string{
			"field": field,
		})
	}
	return digits, nil
}

func normalizeKBK(value string) (string, error) {
	digits, err := normalizeDigitsLen(value, 20, "section1.kbk")
	if err != nil {
		return "", err
	}
	if digits != ndflKBK {
		return "", apperr.Internal("unsupported KBK value for ndfl", nil, map[string]string{
			"field": "section1.kbk",
		})
	}
	return digits, nil
}

func normalizeDigits(value string) string {
	return digitsOnly.ReplaceAllString(strings.TrimSpace(value), "")
}

func normalizeCorrectionNumber(value string) (string, error) {
	num := normalizeDigits(value)
	if num == "" || len(num) > 3 || len(strings.TrimSpace(value)) != len(num) {
		return "", apperr.Internal("invalid correction number for ndfl", nil, map[string]string{
			"field": "header.correction_number",
		})
	}
	return num, nil
}

func normalizeCountryCode(value, field string, index int) (string, error) {
	digits, err := normalizeDigitsLen(value, 3, fmt.Sprintf("%s[%d]", field, index))
	if err != nil {
		return "", err
	}
	return digits, nil
}

func normalizeCurrencyCode(value string, index int) (string, error) {
	return normalizeDigitsLen(value, 3, fmt.Sprintf("appendix2.currency_code[%d]", index))
}

func normalizeIncomeTypeCode(value string, index int) (string, error) {
	return normalizeDigitsLen(value, 4, fmt.Sprintf("appendix2.income_type_code[%d]", index))
}

func normalizeDecimal2(value, field string) (string, error) {
	return normalizeFloat(value, 2, field)
}

func normalizeRate(value, field string) (string, error) {
	return normalizeFloat(value, 9, field)
}

func normalizeFloat(value string, frac int, field string) (string, error) {
	v, err := requiredTrimmed(field, value)
	if err != nil {
		return "", err
	}
	num, parseErr := strconv.ParseFloat(v, 64)
	if parseErr != nil {
		return "", apperr.Internal("invalid decimal value for ndfl field", parseErr, map[string]string{
			"field": field,
		})
	}
	return fmt.Sprintf("%.*f", frac, num), nil
}

func normalizeInt(value, field string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", apperr.Internal("required ndfl integer field is empty", nil, map[string]string{
			"field": field,
		})
	}
	num, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return "", apperr.Internal("invalid integer value for ndfl field", err, map[string]string{
			"field": field,
		})
	}
	return strconv.FormatInt(int64(math.Round(num)), 10), nil
}

func isValidIndividualINN(inn string) bool {
	if len(inn) != 12 {
		return false
	}
	digits := make([]int, 12)
	for i := range len(inn) {
		digits[i] = int(inn[i] - '0')
	}

	check11 := checksumMod11Mod10(digits[:10], []int{7, 2, 4, 10, 3, 5, 9, 4, 6, 8})
	if check11 != digits[10] {
		return false
	}
	check12 := checksumMod11Mod10(digits[:11], []int{3, 7, 2, 4, 10, 3, 5, 9, 4, 6, 8})
	return check12 == digits[11]
}

func checksumMod11Mod10(digits []int, coeffs []int) int {
	sum := 0
	for i := range len(coeffs) {
		sum += digits[i] * coeffs[i]
	}
	return (sum % 11) % 10
}

func taxResidencyToStatus(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "NON_RESIDENT":
		return "2", nil
	case "RESIDENT":
		return "1", nil
	default:
		return "", apperr.Internal("invalid tax residency for ndfl", nil, map[string]string{
			"field": "header.tax_residency",
		})
	}
}

func taxPayerTypeToDocPresenter(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "SOLE_PROPRIETOR":
		return "720", nil
	case "INDIVIDUAL":
		return "760", nil
	default:
		return "", apperr.Internal("invalid taxpayer type for ndfl", nil, map[string]string{
			"field": "header.tax_payer_type",
		})
	}
}

type ndflFileXML struct {
	XMLName        xml.Name        `xml:"Файл"`
	FileID         string          `xml:"ИдФайл,attr"`
	ProgramVersion string          `xml:"ВерсПрог,attr"`
	FormatVersion  string          `xml:"ВерсФорм,attr"`
	Document       ndflDocumentXML `xml:"Документ"`
}

type ndflDocumentXML struct {
	KND              string          `xml:"КНД,attr"`
	DateDoc          string          `xml:"ДатаДок,attr"`
	Period           string          `xml:"Период,attr"`
	ReportYear       int             `xml:"ОтчетГод,attr"`
	TaxOfficeCode    string          `xml:"КодНО,attr"`
	CorrectionNumber string          `xml:"НомКорр,attr"`
	Taxpayer         ndflTaxpayerXML `xml:"СвНП"`
	Signer           ndflSignerXML   `xml:"Подписант"`
	Body             ndflBodyXML     `xml:"НДФЛ3"`
}

type ndflTaxpayerXML struct {
	Person ndflPersonXML `xml:"НПФЛ3"`
}

type ndflPersonXML struct {
	FIO          ndflFIOXML `xml:"ФИОФЛ"`
	INN          string     `xml:"ИННФЛ"`
	DocPresenter string     `xml:"ДокПредст,attr"`
	Status       string     `xml:"Статус,attr"`
	Phone        string     `xml:"Тлф,attr,omitempty"`
}

type ndflFIOXML struct {
	LastName   string `xml:"Фамилия,attr"`
	FirstName  string `xml:"Имя,attr"`
	MiddleName string `xml:"Отчество,attr,omitempty"`
}

type ndflSignerXML struct {
	SignerType string `xml:"ПрПодп,attr"`
}

type ndflBodyXML struct {
	TotalTax  ndflTotalTaxXML   `xml:"СумНалПу"`
	TaxBase   []ndflTaxBaseXML  `xml:"НалБаза"`
	Appendix2 *ndflAppendix2XML `xml:"ДоходИстИно,omitempty"`
	Appendix6 *ndflAppendix6XML `xml:"ИмущНалВычПр,omitempty"`
}

type ndflTotalTaxXML struct {
	Excl227 []ndflTotalTaxLineXML `xml:"СумНалПуИскл227"`
}

type ndflTotalTaxLineXML struct {
	KBK         string `xml:"КБК,attr"`
	OKTMO       string `xml:"ОКТМО,attr"`
	TaxToPay    string `xml:"ПодлУпл,attr"`
	TaxToRefund string `xml:"ПодлВозв,attr"`
}

type ndflTaxBaseXML struct {
	CalcTaxBase     ndflCalcTaxBaseXML  `xml:"РасчНалБаза"`
	CalcTaxToPay    ndflCalcTaxToPayXML `xml:"РасчНалПУ"`
	IncomeGroupCode string              `xml:"ГрупДоход,attr"`
}

type ndflCalcTaxBaseXML struct {
	TotalIncome        string `xml:"СумДох,attr"`
	NonTaxableIncome   string `xml:"СумДохНеНал,attr"`
	TaxableIncome      string `xml:"СумДохНал,attr"`
	Deductions         string `xml:"СумНалВыч,attr"`
	RecognizedExpenses string `xml:"СумРасх,attr"`
	TaxBase            string `xml:"НалБаза,attr"`
}

type ndflCalcTaxToPayXML struct {
	CalculatedTax             string `xml:"Исчисл,attr"`
	WithheldAtSource          string `xml:"Удерж,attr"`
	MaterialBenefitTax        string `xml:"СумУдержМат,attr"`
	TradingFeeCredit          string `xml:"ТСУплПерЗач,attr"`
	FixedAdvanceCredit        string `xml:"СумФиксАван,attr"`
	ForeignTaxCredit          string `xml:"УплИнПодлЗач,attr"`
	PatentTaxCredit           string `xml:"УплПатентЗач,attr"`
	TaxToPay                  string `xml:"ПодлУпл,attr"`
	TaxToRefund               string `xml:"ПодлВозв,attr"`
	SimplifiedDeductionRefund string `xml:"СумВозвУпр,attr"`
}

type ndflAppendix2XML struct {
	Lines []ndflAppendix2LineXML `xml:"РасчДохНалИно"`
}

type ndflAppendix2LineXML struct {
	SourceCountryCode  string `xml:"ОКСМИст,attr"`
	PaymentCountryCode string `xml:"ОКСМЗач,attr"`
	SourceName         string `xml:"НаимИстДох,attr"`
	CurrencyCode       string `xml:"КодВалют,attr"`
	IncomeTypeCode     string `xml:"ВидДоход,attr"`
	IncomeDate         string `xml:"ДатаДох,attr"`
	FXRate             string `xml:"КурсВалютДох,attr"`
	IncomeForeign      string `xml:"ДоходИноВал,attr"`
	IncomeRub          string `xml:"ДоходИноРуб,attr"`
}

type ndflAppendix6XML struct {
	TotalPropertyDeduction string        `xml:"ОбщИмущВыч,attr"`
	OtherPropertySale      ndflSumPr6XML `xml:"ВычПродИмущИн"`
}

type ndflSumPr6XML struct {
	OtherPropertyDeduction      string `xml:"ВычДохПродИмущ,attr,omitempty"`
	OtherPropertyAcquisitionExp string `xml:"РасхПриобрИмущ,attr,omitempty"`
}
