package seed

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
)

const (
	providerBooking          = "booking"
	providerWorkspaceBooking = "workspace_booking"
	providerPayment          = "payment"
	providerUserAccount      = "user_account"
	providerPricing          = "pricing"
)

var requiredProviders = []string{
	providerBooking,
	providerWorkspaceBooking,
	providerPayment,
	providerUserAccount,
	providerPricing,
}

var actionProviderMap = map[string]string{
	action.ActionFindBooking:          providerBooking,
	action.ActionFindWorkspaceBooking: providerWorkspaceBooking,
	action.ActionFindPayment:          providerPayment,
	action.ActionFindUserAccount:      providerUserAccount,
}

type Dataset struct {
	KnowledgeBase     KnowledgeBaseCatalog `json:"-"`
	Bookings          BookingCatalog       `json:"-"`
	WorkspaceBookings WorkspaceCatalog     `json:"-"`
	Payments          PaymentCatalog       `json:"-"`
	Users             UserCatalog          `json:"-"`
	Operators         OperatorCatalog      `json:"-"`
	Providers         ProviderCatalog      `json:"-"`

	knowledgeByKey            map[string]KnowledgeArticle
	bookingByIdentifier       map[string]BookingFixture
	workspaceByIdentifier     map[string]WorkspaceFixture
	paymentByIdentifier       map[string]PaymentFixture
	userByIdentifier          map[string]UserFixture
	providerByName            map[string]ProviderFixture
	providerErrorByIdentifier map[string]map[string]ProviderErrorCase
}

type KnowledgeBaseCatalog struct {
	Articles []KnowledgeArticle `json:"articles"`
}

type KnowledgeArticle struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Category  string `json:"category"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Source    string `json:"source"`
	Version   string `json:"version"`
	UpdatedAt string `json:"updated_at"`
}

type BookingCatalog struct {
	Items []BookingFixture `json:"items"`
}

type BookingFixture struct {
	ID            string   `json:"id"`
	BookingNumber string   `json:"booking_number"`
	Identifiers   []string `json:"identifiers"`
	Service       string   `json:"service"`
	Master        string   `json:"master"`
	Date          string   `json:"date"`
	Time          string   `json:"time"`
	Status        string   `json:"status"`
}

type WorkspaceCatalog struct {
	Items []WorkspaceFixture `json:"items"`
}

type WorkspaceFixture struct {
	ID            string   `json:"id"`
	BookingNumber string   `json:"booking_number"`
	Identifiers   []string `json:"identifiers"`
	WorkspaceType string   `json:"workspace_type"`
	Date          string   `json:"date"`
	Time          string   `json:"time"`
	Duration      string   `json:"duration"`
	Status        string   `json:"status"`
}

type PaymentCatalog struct {
	Items []PaymentFixture `json:"items"`
}

type PaymentFixture struct {
	ID          string   `json:"id"`
	PaymentID   string   `json:"payment_id"`
	Identifiers []string `json:"identifiers"`
	Amount      int      `json:"amount"`
	Date        string   `json:"date"`
	Status      string   `json:"status"`
	Purpose     string   `json:"purpose"`
}

type UserCatalog struct {
	Items []UserFixture `json:"items"`
}

type UserFixture struct {
	ID          string   `json:"id"`
	UserID      string   `json:"user_id"`
	Identifiers []string `json:"identifiers"`
	Email       string   `json:"email"`
	Phone       string   `json:"phone"`
	Status      string   `json:"status"`
}

type OperatorCatalog struct {
	Items []OperatorFixture `json:"items"`
}

type OperatorFixture struct {
	ID         string `json:"id"`
	OperatorID string `json:"operator_id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
}

type ProviderCatalog struct {
	Providers []ProviderFixture `json:"providers"`
}

type ProviderFixture struct {
	Provider            string              `json:"provider"`
	SuccessFixtureIDs   []string            `json:"success_fixture_ids"`
	NotFoundIdentifiers []string            `json:"not_found_identifiers"`
	ErrorCases          []ProviderErrorCase `json:"error_cases"`
}

type ProviderErrorCase struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

type ProviderError struct {
	Provider string
	Code     string
	Message  string
}

func (e ProviderError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func Load(configPath string) (*Dataset, error) {
	seedsDir, err := resolveSeedsDir(configPath)
	if err != nil {
		return nil, err
	}

	dataset := &Dataset{}
	if err := loadJSON(filepath.Join(seedsDir, "knowledge-base.json"), &dataset.KnowledgeBase); err != nil {
		return nil, err
	}
	if err := loadJSON(filepath.Join(seedsDir, "demo-bookings.json"), &dataset.Bookings); err != nil {
		return nil, err
	}
	if err := loadJSON(filepath.Join(seedsDir, "demo-workspace-bookings.json"), &dataset.WorkspaceBookings); err != nil {
		return nil, err
	}
	if err := loadJSON(filepath.Join(seedsDir, "demo-payments.json"), &dataset.Payments); err != nil {
		return nil, err
	}
	if err := loadJSON(filepath.Join(seedsDir, "demo-users.json"), &dataset.Users); err != nil {
		return nil, err
	}
	if err := loadJSON(filepath.Join(seedsDir, "demo-operators.json"), &dataset.Operators); err != nil {
		return nil, err
	}
	if err := loadJSON(filepath.Join(seedsDir, "mock-external-services.json"), &dataset.Providers); err != nil {
		return nil, err
	}
	if err := dataset.buildIndexes(); err != nil {
		return nil, err
	}

	return dataset, nil
}

func (d *Dataset) LookupBooking(identifier string) (map[string]any, error) {
	if errCase, ok := d.providerErrorCase(providerBooking, identifier); ok {
		return nil, ProviderError{Provider: providerBooking, Code: errCase.Code, Message: errCase.Message}
	}

	fixture, ok := d.bookingByIdentifier[normalizeIdentifier(identifier)]
	if !ok {
		return map[string]any{
			"status": "not_found",
			"error":  "booking not found",
		}, nil
	}

	return map[string]any{
		"status":         "found",
		"booking_number": fixture.BookingNumber,
		"service":        fixture.Service,
		"master":         fixture.Master,
		"date":           fixture.Date,
		"time":           fixture.Time,
		"booking_status": fixture.Status,
		"source":         "mock_external",
	}, nil
}

func (d *Dataset) LookupWorkspaceBooking(identifier string) (map[string]any, error) {
	if errCase, ok := d.providerErrorCase(providerWorkspaceBooking, identifier); ok {
		return nil, ProviderError{Provider: providerWorkspaceBooking, Code: errCase.Code, Message: errCase.Message}
	}

	fixture, ok := d.workspaceByIdentifier[normalizeIdentifier(identifier)]
	if !ok {
		return map[string]any{
			"status": "not_found",
			"error":  "workspace booking not found",
		}, nil
	}

	return map[string]any{
		"status":         "found",
		"booking_number": fixture.BookingNumber,
		"workspace_type": fixture.WorkspaceType,
		"date":           fixture.Date,
		"time":           fixture.Time,
		"duration":       fixture.Duration,
		"booking_status": fixture.Status,
		"source":         "mock_external",
	}, nil
}

func (d *Dataset) LookupPayment(identifier string) (map[string]any, error) {
	if errCase, ok := d.providerErrorCase(providerPayment, identifier); ok {
		return nil, ProviderError{Provider: providerPayment, Code: errCase.Code, Message: errCase.Message}
	}

	fixture, ok := d.paymentByIdentifier[normalizeIdentifier(identifier)]
	if !ok {
		return map[string]any{
			"status": "not_found",
			"error":  "payment not found",
		}, nil
	}

	return map[string]any{
		"status":         "found",
		"payment_id":     fixture.PaymentID,
		"amount":         fixture.Amount,
		"date":           fixture.Date,
		"payment_status": fixture.Status,
		"purpose":        fixture.Purpose,
		"source":         "mock_external",
	}, nil
}

func (d *Dataset) LookupUser(identifier string) (map[string]any, error) {
	if errCase, ok := d.providerErrorCase(providerUserAccount, identifier); ok {
		return nil, ProviderError{Provider: providerUserAccount, Code: errCase.Code, Message: errCase.Message}
	}

	fixture, ok := d.userByIdentifier[normalizeUserIdentifier(identifier)]
	if !ok {
		return map[string]any{
			"status": "not_found",
			"error":  "user account not found",
		}, nil
	}

	return map[string]any{
		"status":         "found",
		"user_id":        fixture.UserID,
		"email":          fixture.Email,
		"phone":          fixture.Phone,
		"account_status": fixture.Status,
		"source":         "mock_external",
	}, nil
}

func (d *Dataset) ValidateCatalog(catalog *apppresenter.IntentCatalog) error {
	if catalog == nil {
		return fmt.Errorf("intent catalog is nil")
	}

	errors := make([]string, 0)
	intentKeys := make(map[string]struct{}, len(catalog.Intents))
	for _, intentDef := range catalog.Intents {
		if strings.TrimSpace(intentDef.Key) == "" {
			continue
		}
		intentKeys[intentDef.Key] = struct{}{}
	}

	for _, intentDef := range catalog.Intents {
		if strings.TrimSpace(intentDef.Key) == "" {
			continue
		}
		if len(intentDef.Examples) == 0 {
			errors = append(errors, fmt.Sprintf("intent %s has no examples", intentDef.Key))
		}

		if strings.TrimSpace(intentDef.KnowledgeKey) != "" {
			if _, ok := d.knowledgeByKey[intentDef.KnowledgeKey]; !ok {
				errors = append(errors, fmt.Sprintf("intent %s references missing knowledge_key: %s", intentDef.Key, intentDef.KnowledgeKey))
			}
		}

		if providerName, ok := actionProviderMap[intentDef.Action]; ok {
			providerFixture, exists := d.providerByName[providerName]
			if !exists {
				errors = append(errors, fmt.Sprintf("intent %s references action %s without provider fixtures", intentDef.Key, intentDef.Action))
			} else {
				if len(providerFixture.SuccessFixtureIDs) == 0 {
					errors = append(errors, fmt.Sprintf("provider %s has no success fixtures", providerName))
				}
				if len(providerFixture.NotFoundIdentifiers) == 0 {
					errors = append(errors, fmt.Sprintf("provider %s has no not_found fixtures", providerName))
				}
				if len(providerFixture.ErrorCases) == 0 {
					errors = append(errors, fmt.Sprintf("provider %s has no error fixtures", providerName))
				}
			}
		}

		for _, quickReply := range intentDef.QuickReplies {
			if quickReply.Action != "select_intent" {
				continue
			}
			referencedIntent, _ := quickReply.Payload["intent"].(string)
			if _, ok := intentKeys[referencedIntent]; !ok {
				errors = append(errors, fmt.Sprintf("intent %s quick reply references missing intent: %s", intentDef.Key, referencedIntent))
			}
		}
	}

	for _, providerName := range requiredProviders {
		if _, ok := d.providerByName[providerName]; !ok {
			errors = append(errors, fmt.Sprintf("missing provider fixture set: %s", providerName))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("seed dataset validation failed:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

func (d *Dataset) buildIndexes() error {
	d.knowledgeByKey = make(map[string]KnowledgeArticle, len(d.KnowledgeBase.Articles))
	d.bookingByIdentifier = make(map[string]BookingFixture)
	d.workspaceByIdentifier = make(map[string]WorkspaceFixture)
	d.paymentByIdentifier = make(map[string]PaymentFixture)
	d.userByIdentifier = make(map[string]UserFixture)
	d.providerByName = make(map[string]ProviderFixture, len(d.Providers.Providers))
	d.providerErrorByIdentifier = make(map[string]map[string]ProviderErrorCase, len(d.Providers.Providers))

	errors := make([]string, 0)

	for _, article := range d.KnowledgeBase.Articles {
		if strings.TrimSpace(article.ID) == "" {
			errors = append(errors, "knowledge article has empty id")
			continue
		}
		if strings.TrimSpace(article.Key) == "" {
			errors = append(errors, fmt.Sprintf("knowledge article %s has empty key", article.ID))
			continue
		}
		if _, exists := d.knowledgeByKey[article.Key]; exists {
			errors = append(errors, fmt.Sprintf("duplicate knowledge key: %s", article.Key))
			continue
		}
		d.knowledgeByKey[article.Key] = article
	}

	errors = append(errors, d.indexBookings()...)
	errors = append(errors, d.indexWorkspaceBookings()...)
	errors = append(errors, d.indexPayments()...)
	errors = append(errors, d.indexUsers()...)
	errors = append(errors, d.indexOperators()...)
	errors = append(errors, d.indexProviders()...)

	if len(errors) > 0 {
		return fmt.Errorf("seed dataset index failed:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

func (d *Dataset) indexBookings() []string {
	errors := make([]string, 0)
	seenIDs := make(map[string]struct{}, len(d.Bookings.Items))
	for _, fixture := range d.Bookings.Items {
		if strings.TrimSpace(fixture.ID) == "" {
			errors = append(errors, "booking fixture has empty id")
			continue
		}
		if _, exists := seenIDs[fixture.ID]; exists {
			errors = append(errors, fmt.Sprintf("duplicate booking fixture id: %s", fixture.ID))
			continue
		}
		seenIDs[fixture.ID] = struct{}{}
		for _, identifier := range append([]string{fixture.BookingNumber}, fixture.Identifiers...) {
			key := normalizeIdentifier(identifier)
			if key == "" {
				continue
			}
			if _, exists := d.bookingByIdentifier[key]; exists {
				errors = append(errors, fmt.Sprintf("duplicate booking identifier: %s", identifier))
				continue
			}
			d.bookingByIdentifier[key] = fixture
		}
	}
	return errors
}

func (d *Dataset) indexWorkspaceBookings() []string {
	errors := make([]string, 0)
	seenIDs := make(map[string]struct{}, len(d.WorkspaceBookings.Items))
	for _, fixture := range d.WorkspaceBookings.Items {
		if strings.TrimSpace(fixture.ID) == "" {
			errors = append(errors, "workspace fixture has empty id")
			continue
		}
		if _, exists := seenIDs[fixture.ID]; exists {
			errors = append(errors, fmt.Sprintf("duplicate workspace fixture id: %s", fixture.ID))
			continue
		}
		seenIDs[fixture.ID] = struct{}{}
		for _, identifier := range append([]string{fixture.BookingNumber}, fixture.Identifiers...) {
			key := normalizeIdentifier(identifier)
			if key == "" {
				continue
			}
			if _, exists := d.workspaceByIdentifier[key]; exists {
				errors = append(errors, fmt.Sprintf("duplicate workspace identifier: %s", identifier))
				continue
			}
			d.workspaceByIdentifier[key] = fixture
		}
	}
	return errors
}

func (d *Dataset) indexPayments() []string {
	errors := make([]string, 0)
	seenIDs := make(map[string]struct{}, len(d.Payments.Items))
	for _, fixture := range d.Payments.Items {
		if strings.TrimSpace(fixture.ID) == "" {
			errors = append(errors, "payment fixture has empty id")
			continue
		}
		if _, exists := seenIDs[fixture.ID]; exists {
			errors = append(errors, fmt.Sprintf("duplicate payment fixture id: %s", fixture.ID))
			continue
		}
		seenIDs[fixture.ID] = struct{}{}
		for _, identifier := range append([]string{fixture.PaymentID}, fixture.Identifiers...) {
			key := normalizeIdentifier(identifier)
			if key == "" {
				continue
			}
			if _, exists := d.paymentByIdentifier[key]; exists {
				errors = append(errors, fmt.Sprintf("duplicate payment identifier: %s", identifier))
				continue
			}
			d.paymentByIdentifier[key] = fixture
		}
	}
	return errors
}

func (d *Dataset) indexUsers() []string {
	errors := make([]string, 0)
	seenIDs := make(map[string]struct{}, len(d.Users.Items))
	for _, fixture := range d.Users.Items {
		if strings.TrimSpace(fixture.ID) == "" {
			errors = append(errors, "user fixture has empty id")
			continue
		}
		if _, exists := seenIDs[fixture.ID]; exists {
			errors = append(errors, fmt.Sprintf("duplicate user fixture id: %s", fixture.ID))
			continue
		}
		seenIDs[fixture.ID] = struct{}{}
		for _, identifier := range append([]string{fixture.UserID, fixture.Email, fixture.Phone}, fixture.Identifiers...) {
			key := normalizeUserIdentifier(identifier)
			if key == "" {
				continue
			}
			if _, exists := d.userByIdentifier[key]; exists {
				errors = append(errors, fmt.Sprintf("duplicate user identifier: %s", identifier))
				continue
			}
			d.userByIdentifier[key] = fixture
		}
	}
	return errors
}

func (d *Dataset) indexOperators() []string {
	errors := make([]string, 0)
	seenIDs := make(map[string]struct{}, len(d.Operators.Items))
	seenOperatorIDs := make(map[string]struct{}, len(d.Operators.Items))
	for _, fixture := range d.Operators.Items {
		if strings.TrimSpace(fixture.ID) == "" {
			errors = append(errors, "operator fixture has empty id")
			continue
		}
		if _, exists := seenIDs[fixture.ID]; exists {
			errors = append(errors, fmt.Sprintf("duplicate operator fixture id: %s", fixture.ID))
			continue
		}
		seenIDs[fixture.ID] = struct{}{}
		if strings.TrimSpace(fixture.OperatorID) == "" {
			errors = append(errors, fmt.Sprintf("operator fixture %s has empty operator_id", fixture.ID))
			continue
		}
		if _, exists := seenOperatorIDs[fixture.OperatorID]; exists {
			errors = append(errors, fmt.Sprintf("duplicate operator_id: %s", fixture.OperatorID))
			continue
		}
		seenOperatorIDs[fixture.OperatorID] = struct{}{}
	}
	return errors
}

func (d *Dataset) indexProviders() []string {
	errors := make([]string, 0)
	for _, fixture := range d.Providers.Providers {
		if strings.TrimSpace(fixture.Provider) == "" {
			errors = append(errors, "provider fixture has empty provider name")
			continue
		}
		if _, exists := d.providerByName[fixture.Provider]; exists {
			errors = append(errors, fmt.Sprintf("duplicate provider fixture set: %s", fixture.Provider))
			continue
		}
		d.providerByName[fixture.Provider] = fixture
		errorIndex := make(map[string]ProviderErrorCase, len(fixture.ErrorCases))
		seenErrorIDs := make(map[string]struct{}, len(fixture.ErrorCases))
		for _, errCase := range fixture.ErrorCases {
			if strings.TrimSpace(errCase.ID) == "" {
				errors = append(errors, fmt.Sprintf("provider %s has error case with empty id", fixture.Provider))
				continue
			}
			if _, exists := seenErrorIDs[errCase.ID]; exists {
				errors = append(errors, fmt.Sprintf("provider %s has duplicate error case id: %s", fixture.Provider, errCase.ID))
				continue
			}
			seenErrorIDs[errCase.ID] = struct{}{}

			key := normalizeProviderIdentifier(fixture.Provider, errCase.Identifier)
			if key == "" {
				errors = append(errors, fmt.Sprintf("provider %s has error case with empty identifier", fixture.Provider))
				continue
			}
			if _, exists := errorIndex[key]; exists {
				errors = append(errors, fmt.Sprintf("provider %s has duplicate error identifier: %s", fixture.Provider, errCase.Identifier))
				continue
			}
			errorIndex[key] = errCase
		}
		d.providerErrorByIdentifier[fixture.Provider] = errorIndex

		for _, successID := range fixture.SuccessFixtureIDs {
			if !d.hasSuccessFixture(fixture.Provider, successID) {
				errors = append(errors, fmt.Sprintf("provider %s references missing success fixture id: %s", fixture.Provider, successID))
			}
		}
	}
	return errors
}

func (d *Dataset) hasSuccessFixture(providerName, fixtureID string) bool {
	switch providerName {
	case providerBooking:
		for _, item := range d.Bookings.Items {
			if item.ID == fixtureID {
				return true
			}
		}
	case providerWorkspaceBooking:
		for _, item := range d.WorkspaceBookings.Items {
			if item.ID == fixtureID {
				return true
			}
		}
	case providerPayment:
		for _, item := range d.Payments.Items {
			if item.ID == fixtureID {
				return true
			}
		}
	case providerUserAccount:
		for _, item := range d.Users.Items {
			if item.ID == fixtureID {
				return true
			}
		}
	case providerPricing:
		for _, article := range d.KnowledgeBase.Articles {
			if article.ID == fixtureID {
				return true
			}
		}
	}
	return false
}

func (d *Dataset) providerErrorCase(providerName, identifier string) (ProviderErrorCase, bool) {
	providerErrors, ok := d.providerErrorByIdentifier[providerName]
	if !ok {
		return ProviderErrorCase{}, false
	}
	errCase, ok := providerErrors[normalizeProviderIdentifier(providerName, identifier)]
	return errCase, ok
}

func resolveSeedsDir(configPath string) (string, error) {
	dir := filepath.Clean(configPath)
	if absDir, err := filepath.Abs(dir); err == nil {
		dir = absDir
	}
	if dir == "." || dir == "" {
		if cwd, err := os.Getwd(); err == nil {
			dir = cwd
		}
	}

	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "seeds")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("failed to locate seeds directory from config path %q", configPath)
}

func loadJSON(path string, dest any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func normalizeIdentifier(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeUserIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "@") {
		return strings.ToLower(value)
	}
	digits := make([]rune, 0, len(value))
	for _, r := range value {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	if len(digits) >= 10 {
		return string(digits)
	}
	return strings.ToUpper(value)
}

func normalizeProviderIdentifier(providerName, identifier string) string {
	if providerName == providerUserAccount {
		return normalizeUserIdentifier(identifier)
	}
	return normalizeIdentifier(identifier)
}
