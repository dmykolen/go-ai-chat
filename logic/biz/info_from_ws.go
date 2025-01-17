package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
	"gitlab.dev.ict/golang/libs/ws"
	"gitlab.dev.ict/golang/libs/ws/cimws"
	"gitlab.dev.ict/golang/libs/ws/omws"
)

var ErrEmptyResponseOrHasErrors = errors.New("empty response from external system or has errors")

const (
	ch            = "biz-info"
	UseCommonMain = "USE_COMMON_MAIN"
	Konsultant    = "KA_CONSULTANT"
	SalesExpert   = "SALES_EXPERT"

	FMC_VOIP   = "FMC_GET_VOIP"
	FMC_PREFIX = "FMC_GET_PREFIX"
	FMC_MOBILE = "FMC_GET_MOBILE"
)

type WSGetter struct {
	log   *gologgers.Logger
	cimws *cimws.CimClient
	omws  *omws.Client
}

func NewWSGetter(log *gologgers.Logger, cimws *cimws.CimClient, omws *omws.Client) *WSGetter {
	log.Info("Creating new WSGetter(clients logic to call webservices) instance")
	return &WSGetter{log: log, cimws: cimws, omws: omws}
}

// GetAccount returns account info by msisdn (9 profiles)
func (g *WSGetter) GetAccount(ctx context.Context, msisdn string) (acc *Account, err error) {
	log := g.log.RecWithCtx(ctx, ch)
	now := time.Now()
	entryMsisdn := ws.E{Key: ws.MSISDN, Value: msisdn}
	res, err := cimws.RESTCallGet[cimws.CustomerAccount](g.cimws, ctx, cimws.P_GetDataLight, "WEB", entryMsisdn)
	if err != nil {
		log.Errorf("Error fetching customer account by msisdn[%s] from cimws => %v", msisdn, err)
		return nil, err
	}

	if !res.IsSuccessfulAndNotEmpty() {
		log.Warnf("Customer acc is empty or response has errors! cimws response: %v", res.String(false))
		return nil, ErrEmptyResponseOrHasErrors
	}

	acc = NewAccountFromCA(res.GetEntity())
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if acc.ContractNo != "" {
			acc.Contract = g.GetContract(ctx, acc.ContractNo)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err = cimws.RESTCallGet[cimws.CustomerAccount](g.cimws, ctx, cimws.P_GetAccFull, "WEB", ws.E{Key: ws.AccID, Value: acc.BillingID})
		if err != nil {
			log.Errorf("cimws.RESTCallGet[cimws.Account](%s) => %v", acc.BillingID, err)
			return
		}

		if !res.IsSuccessfulAndNotEmpty() {
			log.Warnf("cimws response is unsuccessfull or empty for profile[%s] and accountId[%v]", cimws.P_GetAccFull, acc.BillingID)
			return
		}

		defer acc.Unlock()
		acc.Lock()
		acc.populateBalances(res.GetEntity())
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		r, e := omws.RESTCallGet[omws.Product](g.omws, ctx, omws.P_GetProudctsActShort, "WEB", entryMsisdn)
		if e != nil {
			log.Errorf("omws error: %v", e)
			return
		}
		if !r.IsSuccessfulAndNotEmpty() {
			log.Errorf("omws.RESTCallGet[omws.Product](%s) => %v", msisdn, e)
			return
		}

		defer acc.Unlock()
		acc.Lock()
		acc.Services = populateServices(r)
		acc.UseCommonMain = acc.Services.Find(UseCommonMain).IsAct()
	}()

	wg.Wait()

	acc.IfEnoughMoney = IfEnoughMoney(log, acc, acc.Contract)
	log.Infof("Account fetched! Elapsed=%v TotalProfiles=9", time.Since(now))
	return
}

// GetContract returns contract info by contract number (3 profiles)
func (g *WSGetter) GetContract(ctx context.Context, conNo string) *Contract {
	log := g.log.RecWithCtx(ctx, ch)
	contract := &Contract{
		Mutex:    &sync.Mutex{},
		No:       conNo,
		Managers: map[string]interface{}{},
	}
	wg := sync.WaitGroup{}

	wg.Add(3)
	go func() {
		defer wg.Done()

		res, err := cimws.RESTCallGet[cimws.Agreement](g.cimws, ctx, cimws.P_WebNew2002, "A", ws.E{Key: ws.ConNo, Value: conNo})
		if err != nil {
			log.Error(err)
			return
		}

		defer contract.Unlock()
		contract.Lock()
		contract.populateBalancesContract(res.GetEntity().GetAgreementOwner().GetAccFromAgreementOwner())
		contract.BillingID = GetAgrAccId(res.GetEntity())
	}()

	go func() {
		defer wg.Done()

		res, err := cimws.RESTCallGet[cimws.Agreement](g.cimws, ctx, cimws.P_GetContactPersonsContract, "A", ws.E{Key: ws.ConNo, Value: conNo})
		if err != nil {
			log.Error(err)
			return
		}

		defer contract.Unlock()
		contract.Lock()
		contract.ContactPersons = NewContactPersons(res.GetEntity())
	}()

	go func() {
		defer wg.Done()

		res, err := omws.RESTCallGet1[omws.ProductOffering](g.omws, ctx, ws.BuildRequestOmExt(omws.P_GetProudctAvailable, "WEB", "", []string{Konsultant, SalesExpert}, ws.E{ws.AgrCode, conNo}))
		if err != nil {
			log.Error(err)
			return
		}

		defer contract.Unlock()
		contract.Lock()
		contract.Managers[Konsultant] = strings.Replace(omws.FindProduct(res.Items, Konsultant).GetExtraFieldVal(), "astelit.ukr", "lifecell.com.ua", 1)
		contract.Managers[SalesExpert] = strings.Replace(omws.FindProduct(res.Items, SalesExpert).GetExtraFieldVal(), "astelit.ukr", "lifecell.com.ua", 1)
	}()

	wg.Wait()
	return contract
}

func (g *WSGetter) GetFMC_VOIP(ctx context.Context, contractNo, msisdn string) (r *WSResponse, err error) {
	log := g.log.RecWithCtx(ctx, ch)

	r, err = g.getFMCSettings(log, FMC_VOIP, contractNo, msisdn)
	if err != nil {
		return
	}

	if msisdn != "" {
		gr := r.SearchGroupByMSIDN(msisdn)
		if gr != nil {
			gr.PhoneNumbers = map[string]PhoneInfo{msisdn: (*gr).PhoneNumbers[msisdn]}

			return &WSResponse{fmt.Sprintf("%d_%d", gr.GroupID, gr.IPAuthGroupID): *gr}, nil
		} else {
			return nil, ErrEmptyResponseOrHasErrors
		}
	}

	return
}

func (g *WSGetter) GetFMC_MOBILE(ctx context.Context, contractNo, msisdn string) (r *WSResponse, err error) {
	log := g.log.RecWithCtx(ctx, ch)

	// var prefixResp *WSResponse
	var mobileResp *WSResponse

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		r, err = g.getFMCSettings(log, FMC_PREFIX, contractNo, msisdn)
		log.WithData(gologgers.M{"err": err}).Infof("PREFIX_RESP %s", utils.Json(mobileResp))
	}()

	go func() {
		defer wg.Done()
		mobileResp, err = g.getFMCSettings(log, FMC_MOBILE, contractNo, msisdn)
		log.Infof("MOBILE_RESP %s", utils.Json(mobileResp))
	}()

	wg.Wait()

	if err != nil {
		return
	}

	for k, v := range *mobileResp {
		log.Infof("KEY=%s GR=%v PhoneNumbers-%v", k, r.Get(k), v.PhoneNumbers)
		r.UpdateGroupPhoneNumbers(k, v.PhoneNumbers)
	}

	if msisdn != "" {
		gr := r.SearchGroupByMSIDN(msisdn)
		if gr != nil {
			return &WSResponse{fmt.Sprintf("%d_%d", gr.GroupID, gr.IPAuthGroupID): *gr}, nil
		} else {
			return nil, ErrEmptyResponseOrHasErrors
		}
	}

	return
}

func (g *WSGetter) GetFMC(ctx context.Context, contractNo, msisdn string) (*AggregatedPhoneInfo, error) {
	log := g.log.RecWithCtx(ctx, ch)

	profiles := []string{FMC_VOIP, FMC_PREFIX, FMC_MOBILE}

	wg := sync.WaitGroup{}
	wg.Add(len(profiles))

	var fmcInfo []*AggregatedPhoneInfo
	var err error

	for _, profile := range profiles {
		go func(profile string) {
			defer wg.Done()

			phoneData, e := g.getFMC(log, profile, contractNo, msisdn)
			if e != nil {
				err = e
				return
			}

			fmcInfo = append(fmcInfo, phoneData)

			// defer phoneData.Unlock()
			// phoneData.Lock()

			// if fmcInfo == nil {
			// 	fmcInfo = phoneData
			// } else {
			// 	fmcInfo.Merge(phoneData)
			// }
		}(profile)
	}

	wg.Wait()

	if err != nil {
		return nil, err
	}

	return fmcInfo[0], nil
}

func (g *WSGetter) getFMCSettings(log *gologgers.LogRec, profile, contractNo, msisdn string) (r *WSResponse, err error) {
	res, err := cimws.RESTCallOperate(g.cimws, log.Ctx, cimws.CreateRequestOperateAcc(profile, "Iguana", "IUI", cimws.KV{"contractCode": contractNo}))
	if err != nil {
		log.Errorf("(%s) => %v", msisdn, err)
		return nil, err
	}

	if !res.IsSuccessfulAndNotEmpty() {
		log.Warnf("FMC data is empty or response has errors! cimws response: %v", res)
		return nil, ErrEmptyResponseOrHasErrors
	}

	settings := res.ChainResults[0].GetVarValue("settings")

	var response WSResponse
	err = json.Unmarshal([]byte(settings), &response)
	if err != nil {
		log.Errorf("json.Unmarshal error: %v", err)
		return nil, err
	}

	if len(response) == 0 {
		log.Warnf("No FMC data found for phone number: %s", msisdn)
		return nil, ErrEmptyResponseOrHasErrors
	}
	return &response, nil
}

func (g *WSGetter) getFMC(log *gologgers.LogRec, profile, contractNo, msisdn string) (*AggregatedPhoneInfo, error) {
	res, err := cimws.RESTCallOperate(g.cimws, log.Ctx, cimws.CreateRequestOperateAcc(profile, "Iguana", "IUI", cimws.KV{"contractCode": contractNo}))
	if err != nil {
		log.Errorf("(%s) => %v", msisdn, err)
		return nil, err
	}

	if !res.IsSuccessfulAndNotEmpty() {
		log.Warnf("FMC data is empty or response has errors! cimws response: %v", res)
		return nil, ErrEmptyResponseOrHasErrors
	}

	settings := res.ChainResults[0].GetVarValue("settings")

	var response WSResponse
	err = json.Unmarshal([]byte(settings), &response)
	if err != nil {
		log.Errorf("json.Unmarshal error: %v", err)
		return nil, err
	}

	if len(response) == 0 {
		log.Warnf("No FMC data found for phone number: %s", msisdn)
		return nil, ErrEmptyResponseOrHasErrors
	}

	phoneData := aggregatePhoneInfo(response, msisdn)
	if phoneData == nil {
		log.Warnf("No FMC data found for phone number: %s", msisdn)
		return nil, ErrEmptyResponseOrHasErrors
	}

	return phoneData, nil
}

func changeEmailDomain(email string) string {
	if email == "" {
		return ""
	}

	return strings.Replace(email, "astelit.ukr", "lifecell.com.ua", 1)
}

// If enough money on account
func IfEnoughMoney(log *gologgers.LogRec, acc *Account, contract *Contract) bool {
	log.Debugf("acc.Balances: %s", utils.Json(acc.Balances))
	log.Debugf("Contract.Balances: %s, CreditLimit: %v", utils.Json(contract.Balances), contract.CreditLimit)

	if lm := acc.Balances["Line_Main"]; lm != nil && *lm > 0 {
		return true
	}
	if !acc.UseCommonMain {
		return false
	}
	l_sm := acc.Balances["Line_SpendingLimit"]
	l_cm_u := acc.Balances["Line_CM_Usage"]
	if l_sm != nil && l_cm_u != nil && *l_sm < *l_cm_u {
		if cm, exists := contract.Balances["Common_Main"]; exists && cm != nil {
			return (*cm + contract.CreditLimit) > 0
		}
	}

	return false
}

func populateServices(resp *ws.GetResponseREST[omws.Product]) (services []Service) {
	if !resp.IsSuccessfulAndNotEmpty() {
		return nil
	}
	for _, p := range resp.Items {
		services = append(services, Service{p.Name, p.ProductOffering.ID, p.ProductStatus, p.ProductOffering.Description})
	}
	return
}

func NewAccountFromCA(ca *cimws.CustomerAccount) *Account {
	specAccLight := ca.CustomerBilling.CustomerBillingDescBy.Get("CustomerSubscriberAccountLightSpec")
	specSubsbAccRole := ca.CustomerAccountInteractions[0].BusinessIntRoleDescrBySpec.Get("SubscriberAccountRoleSpec")
	return &Account{
		Mutex:         &sync.Mutex{},
		BillingID:     ca.ID,
		Status:        ca.AccountStatus,
		Tariff:        okValue(ws.GetValueGeneric[string](specAccLight, "tariffName")),
		Msisdn:        okValue(ws.GetValueGeneric[string](specAccLight, "msisdn")),
		ContractNo:    okValue(ws.GetValueGeneric[string](specAccLight, "contractNo")),
		UseCommonMain: okValue(ws.GetValueGeneric[bool](specSubsbAccRole, "specSubsbAccRole")),
	}
}

func NewContactPersons(p *cimws.Agreement) (persons []*ContactPerson) {
	for _, v := range p.GetContacts() {
		cp := &ContactPerson{
			FIO:         v.Party.IndividualsName[0],
			Birth:       v.Party.AliveDuring.StartDateTime,
			ContactType: v.TypeCode,
			Position:    v.Position,
		}
		cp.Email, cp.Phone, cp.PhoneLifecell, cp.LineMsisdn = cimws.GetParamsFromLogicalAddr(v.PartyRoleContactableVia)
		persons = append(persons, cp)
	}
	return
}
