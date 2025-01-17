package biz

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"gitlab.dev.ict/golang/libs/utils"
	"gitlab.dev.ict/golang/libs/ws"
	"gitlab.dev.ict/golang/libs/ws/cimws"
	"gitlab.dev.ict/golang/libs/ws/omws"
)

type Account struct {
	*sync.Mutex
	BillingID     string               `json:"billingId"`
	Status        string               `json:"statusCode"`
	Tariff        string               `json:"tariffCode"`
	Msisdn        string               `json:"msisdn"`
	ContractNo    string               `json:"contractNo"`
	UseCommonMain bool                 `json:"useCommonMain"` // USE_COMMON_MAIN
	IfEnoughMoney bool                 `json:"ifEnoughMoney"`
	Balances      map[string]*float64  `json:"balances"` // Line_Main, Line_SpendingLimit, Line_CM_Usage
	Services      Services             `json:"services"`
	Contract      *Contract            `json:"contract"`
	FMCVoip       *AggregatedPhoneInfo `json:"fmcVoip,omitempty"`
}

func (a *Account) String(indent ...bool) string { return ws.StringObj(a, indent...) }

func (a *Account) populateBalances(e *cimws.CustomerAccount) {
	a.Balances = map[string]*float64{
		"Line_Main":          accBalVal(e, "Line_Main"),
		"Line_SpendingLimit": accBalVal(e, "Line_SpendingLimit"),
		"Line_CM_Usage":      accBalVal(e, "Line_CM_Usage"),
	}
}

type Contract struct {
	*sync.Mutex
	BillingID      string
	No             string
	Balances       map[string]*float64    // Common_Main, Common_Debt, Common_Due
	CreditLimit    float64                // GET_CREDIT_LIMIT_INFO (CREDIT_LIMIT_CON з srv.extra_fields)
	Managers       map[string]interface{} // GET_PRODUCTS_ACTIVE_CONTRACT_IUI (productOfferingId: SALES_EXPERT, KA_CONSULTANT)
	ContactPersons []*ContactPerson       // GET_CONTRACT_CONTACT_EMAIL + GET_CONTRACT_CONTACT_PERSONS
}

func (c *Contract) String(indent ...bool) string { return ws.StringObj(c, indent...) }

func (c *Contract) populateBalancesContract(e *cimws.CustomerAccount) {
	c.CreditLimit = e.CustomerBilling.CreditLimit
	c.Balances = map[string]*float64{
		"Common_Main": accBalVal(e, "Common_Main"),
		"Common_Debt": accBalVal(e, "Common_Debt"),
	}
}

type ContactPerson struct {
	FIO                                                            cimws.IndividualName
	Birth                                                          int64
	Email, Phone, PhoneLifecell, LineMsisdn, ContactType, Position string
}

type Service struct {
	Name, Id, Status, Desc string
}

func (s *Service) String() string {
	return utils.JsonStr(s)
}

func (s *Service) IsAct() bool {
	return s != nil && s.Id != "" && s.Status == omws.ACT
}

type Services []Service

func (s *Services) Find(id string) *Service {
	for _, i := range *s {
		if i.Id == id {
			return &i
		}
	}
	return &Service{}
}

// ################################################################################
// ################################################################################

type WSResponse map[string]GroupInfo

func (wsr *WSResponse) Get(key string) *GroupInfo {
	if group, exists := (*wsr)[key]; exists {
		return &group
	}
	return nil
}

func (wsr *WSResponse) SearchPhoneInfo(msisdn string) *PhoneInfo {
	for _, group := range *wsr {
		if phone, exists := group.PhoneNumbers[msisdn]; exists {
			return &phone
		}
	}
	return nil
}

func (wsr *WSResponse) SearchGroupByMSIDN(msisdn string) *GroupInfo {
	for _, group := range *wsr {
		for k := range group.PhoneNumbers {
			if k == msisdn {
				return &group
			}
		}
	}
	return nil
}

// UpdatePhoneInfoByNumber updates a single phone's info in the WSResponse
// Returns true if update was successful, false if phone number wasn't found
func (wsr *WSResponse) UpdatePhoneInfoByNumber(msisdn string, newInfo PhoneInfo) bool {
	if wsr == nil {
		return false
	}

	for groupKey, group := range *wsr {
		if _, exists := group.PhoneNumbers[msisdn]; exists {
			group.PhoneNumbers[msisdn] = newInfo
			(*wsr)[groupKey] = group // Update the group in the map
			return true
		}
	}
	return false
}

// UpdateGroupPhoneNumbers updates all phone numbers for a specific group
// Returns true if update successful, false if group not found or WSResponse is nil
func (wsr *WSResponse) UpdateGroupPhoneNumbers(groupKey string, newPhoneNumbers map[string]PhoneInfo) bool {
	if wsr == nil {
		return false
	}

	if group, exists := (*wsr)[groupKey]; exists {
		group.PhoneNumbers = newPhoneNumbers
		(*wsr)[groupKey] = group // Update the group in the map
		return true
	}

	return false
}

type GroupInfo struct {
	GroupID          int                  `json:"group_id"`
	IPAuthGroupID    int                  `json:"ip_auth_grp_id"`
	Prefix           string               `json:"prefix"`
	BillingContract  string               `json:"billing_contract"`
	Password         string               `json:"password"`
	Security         string               `json:"security"`
	ParallelCalls    int                  `json:"parallelCalls,omitempty"`
	OExpires         string               `json:"oExpires,omitempty"`
	MOConnectionType string               `json:"moConnectionType"`
	MoIp             string               `json:"moIp"`
	MTConnectionType string               `json:"mtConnectionType"`
	MtIp             string               `json:"mtIp"`
	PhoneNumbers     map[string]PhoneInfo `json:"phoneNumbers"`
}

type PhoneInfo struct {
	XForward     Bool   `json:"xFORWARD"`
	IsLocked     Bool   `json:"is_locked"`
	IntLocked    Bool   `json:"int_locked"`
	LFwdOriginal string `json:"lFwdOriginal"`
	TariffPlan   string `json:"tariffPlan"`
}

type Bool bool

func (b *Bool) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch v := v.(type) {
	case float64:
		*b = Bool(v != 0)
	case bool:
		*b = Bool(v)
	default:
		return fmt.Errorf("invalid boolean value")
	}
	return nil
}

func printFMCResponse(response WSResponse) {
	for key, groupInfo := range response {
		fmt.Printf("Group: %s\n", key)
		fmt.Printf("  Billing Contract: %s\n", groupInfo.BillingContract)
		fmt.Printf("  Parallel Calls: %d\n", groupInfo.ParallelCalls)
		fmt.Printf("  MO Connection Type: %s\n", groupInfo.MOConnectionType)
		fmt.Printf("  MO IP: %s\n", groupInfo.MoIp)
		fmt.Printf("  MT Connection Type: %s\n", groupInfo.MTConnectionType)
		fmt.Printf("  MT IP: %s\n", groupInfo.MtIp)
		fmt.Printf("  Security: %s\n", groupInfo.Security)

		for phoneNumber, numberInfo := range groupInfo.PhoneNumbers {
			fmt.Printf("  Phone: %s\n", phoneNumber)
			fmt.Printf("    Tariff Plan: %s\n", numberInfo.TariffPlan)
			fmt.Printf("    XForward: %v\n", numberInfo.XForward)
			fmt.Printf("    IsLocked: %v\n", numberInfo.IsLocked)
			fmt.Printf("    IntLocked: %v\n", numberInfo.IntLocked)
			fmt.Printf("    LFwdOriginal: %s\n", numberInfo.LFwdOriginal)
		}
	}
}

type AggregatedPhoneInfo struct {
	PhoneNumber      string   `json:"phoneNumber"`      // The specific phone number
	MOConnectionType string   `json:"moConnectionType"` // Тип підключення для вихідних дзвінків (pass_auth or ip_auth)
	MoIp             []string `json:"moIp"`             // IP list, only if MoConnectionType is "ip_auth"
	MTConnectionType string   `json:"mtConnectionType"` // Тип підключення для вхідних дзвінків (register_type or ip_type)
	MtIp             []string `json:"mtIp"`             // IP list, only if MtConnectionType is "ip_type"
	ParallelCalls    int      `json:"parallelCalls"`    // кількість одночасних викликів
	Password         string   `json:"password"`         // пароль
	Security         string   `json:"security"`         // тип безпеки: tsl_rtp або tsl_srtp
	XForward         bool     `json:"xFORWARD"`         // =1 якщо доволена переадресація для FMC схеми
	IsLocked         bool     `json:"is_locked"`        // =1 якщо номер заблокований на платформі
	IntLocked        bool     `json:"int_locked"`       // =1 якщо заблоковані міжнародні виклики
	TariffPlan       string   `json:"tariffPlan"`       // Тарифний план
}

func (api AggregatedPhoneInfo) String() string {
	return fmt.Sprintf(
		"PhoneNumber: %s\n\tMOConnectionType: %s\n\tMoIp: [%s]\n\tMTConnectionType: %s\n\tMtIp: [%s]\n\tParallelCalls: %d\n\tPassword: %s\n\tSecurity: %s\n\tXForward: %t\n\tIsLocked: %t\n\tIntLocked: %t\n\tTariffPlan: %s\n",
		api.PhoneNumber,
		api.MOConnectionType,
		joinStrings(api.MoIp),
		api.MTConnectionType,
		joinStrings(api.MtIp),
		api.ParallelCalls,
		api.Password,
		api.Security,
		api.XForward,
		api.IsLocked,
		api.IntLocked,
		api.TariffPlan,
	)
}

func aggregatePhoneInfo(wsResp WSResponse, phoneNumber string) *AggregatedPhoneInfo {
	for _, group := range wsResp {
		for number, phone := range group.PhoneNumbers {
			if number == phoneNumber {
				// Prepare IP lists based on connection types
				var moIp []string
				if group.MOConnectionType == "ip_auth" {
					moIp = append(moIp, group.MoIp)
				}

				var mtIp []string
				if group.MTConnectionType == "ip_type" {
					mtIp = append(mtIp, group.MtIp)
				}

				// Return aggregated phone info
				return &AggregatedPhoneInfo{
					PhoneNumber:      number,
					MOConnectionType: group.MOConnectionType,
					MoIp:             moIp,
					MTConnectionType: group.MTConnectionType,
					MtIp:             mtIp,
					ParallelCalls:    group.ParallelCalls,
					Password:         group.Password,
					Security:         group.Security,
					XForward:         bool(phone.XForward),
					IsLocked:         bool(phone.IsLocked),
					IntLocked:        bool(phone.IntLocked),
					TariffPlan:       phone.TariffPlan,
				}
			}
		}
	}
	// Return nil if phone number is not found
	return nil
}

func joinStrings(slice []string) string {
	if len(slice) == 0 {
		return ""
	}
	return strings.Join(slice, ", ")
}
