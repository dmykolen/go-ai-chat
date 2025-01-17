package biz

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/olekukonko/tablewriter"
	"github.com/stretchr/testify/assert"
	"gitlab.dev.ict/golang/libs/gologgers"
	gl "gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
	"gitlab.dev.ict/golang/libs/ws"
	"gitlab.dev.ict/golang/libs/ws/cimws"
	"gitlab.dev.ict/golang/libs/ws/omws"
	"golang.org/x/exp/maps"
)

const (
	conNo1    = "300070023"
	conAcc1   = 3833316
	devCimWs  = "http://dev-main-tm-cim-1.dev.ict:8080"
	testCimWs = "http://dev-test-tm-cim-1.dev.ict:8080"
)

var (
	l = gl.New(gl.WithChannel("wsGetter"), gl.WithLevel("trace"), gl.WithOC())
	// cimwsClient = cimws.NewClient(cimws.WithParams(&cimws.ApiParamas{Url: "http://dev-main-tm-cim-1.dev.ict:8080", Username: "iguana", Password: "iguana", TO: 10, IsDebug: false}), cimws.WithLogger(l.Logger))
	// wsGetter = NewWSGetter(l, cimws.NewClient(cimws.WithParams(&cimws.ApiParamas{Url: "http://dev-main-tm-cim-1.dev.ict:8080", Username: "iguana", Password: "iguana", TO: 10, IsDebug: false}), cimws.WithLogger(l.Logger)), nil)
	wsGetter *WSGetter
	ctx      = utils.GenerateCtxWithRid()

	omwsApiParams  = &omws.ApiParamas{}
	cimwsApiParams = &cimws.ApiParamas{}
)

func init() {
	env.Parse(omwsApiParams)
	env.Parse(cimwsApiParams)
	l.Info(utils.JsonPrettyStr(omwsApiParams))
	l.Info(utils.JsonPrettyStr(cimwsApiParams.DebugOFF().URL(testCimWs)))
	wsGetter = NewWSGetter(l,
		cimws.NewClient(cimws.WithParams(cimwsApiParams.URL(devCimWs)), cimws.WithLogger(l.Logger)),
		omws.NewClient(omws.WithParams(omwsApiParams), omws.WithLogger(l)))
}

func Test_init(t *testing.T) {
	t.Log(utils.JsonPrettyStr(cimwsApiParams))
	cimwsApiParams.DebugOFF().URL("http://google.com")
	t.Log(utils.JsonPrettyStr(cimwsApiParams))

	t.Run("Tratattta", func(t *testing.T) {
		t.Log(strings.Split(t.Name(), "/")[1])

		dict := map[string]int{
			"VOIP":   0,
			"MOBILE": 1,
			"PREFIX": 2,
		}

		t.Log(maps.Keys(dict))
	})
}

func Test_22(t *testing.T) {
	t.Log(changeEmailDomain("marianna.sukhova@astelit.ukr"))
	t.Logf("[%s]", changeEmailDomain(""))
	t.Logf("[%s]", changeEmailDomain("marianna.sukhova@--astelit.ukr"))
}

func TestInfoFromWS_GetInfoFromWS2(t *testing.T) {
	res, err := cimws.RESTCallGet[cimws.Agreement](wsGetter.cimws, ctx, cimws.P_GetAgrFull, "A", ws.E{Key: ws.AccID, Value: conAcc1})
	assert_response(t, res, err)
	t.Log(res.GetEntity().String(true))
}

func TestInfoFromWS_GetAccount(t *testing.T) {
	l.Info(utils.JsonPrettyStr(cimwsApiParams.DebugOFF().URL(devCimWs)))
	cimwsCl := cimws.NewClient(cimws.WithParams(cimwsApiParams), cimws.WithLogger(l.Logger))
	wsGetter := NewWSGetter(l, cimwsCl, omws.NewClient(omws.WithParams(omwsApiParams), omws.WithLogger(l)))

	testCases := []struct {
		name, msisdn string
	}{
		{"TestInfoFromWS_GetInfoFromWS_Success", "380930164453"},
		{"TestInfoFromWS_GetInfoFromWS_Error", "380930164400"},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			acc, err := wsGetter.GetAccount(ctx, tt.msisdn)

			if tt.name == "TestInfoFromWS_GetInfoFromWS_Error" {
				assert.Error(t, err)
				assert.Empty(t, acc)
				return
			}

			assert.NoError(t, err)
			if assert.NotNil(t, acc) {
				t.Logf("ACC_INFO=>%s", acc.String(true))
				t.Logf("USE_COMMON_MAIN: [isAct=%t] => %s", acc.Services.Find("USE_COMMON_MAIN").IsAct(), acc.Services.Find("USE_COMMON_MAIN"))
			}
		})
	}
}

func TestInfoFromWS_GetData(t *testing.T) {
	var err error
	var acc *Account
	var agr *Contract

	t.Run("GetACC", func(t *testing.T) {
		t.Log("--")
		acc, err = wsGetter.GetAccount(ctx, "380930164453")
		assert.NoError(t, err)
		assert.NotEmpty(t, acc)
		t.Log(acc.String(false))
	})

	t.Run("GetACC", func(t *testing.T) {
		t.Log("k33")
		acc, err = wsGetter.GetAccount(ctx, "380933780678")
		assert.NoError(t, err)
		assert.NotEmpty(t, acc)
		t.Log(acc.String(false))
	})

	t.Run("GetAGREEMENT", func(t *testing.T) {
		agr = wsGetter.GetContract(ctx, conNo1)
		assert.NotEmpty(t, agr)
		t.Log(agr.String(true))
	})

	// t.Log("IfEnoughMoney ===>", IfEnoughMoney(l.Rec(), acc, agr))
}

func TestInfoFromWS_GetInfoFromWS(t *testing.T) {
	tests := []struct {
		name, msidn string
	}{
		{"TestInfoFromWS_GetInfoFromWS", "380930164453"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acc, err := wsGetter.GetAccount(ctx, tt.msidn)
			assert.NoError(t, err)
			if assert.NotNil(t, acc) {
				t.Logf("ACC_INFO=>%s", acc.String(true))
				t.Logf("USE_COMMON_MAIN: [isAct=%t] => %s", acc.Services.Find("USE_COMMON_MAIN").IsAct(), acc.Services.Find("USE_COMMON_MAIN"))
			}
		})
	}
}

func Test_GetAgr(t *testing.T) {
	tests := []struct {
		name, key, val string
	}{
		{"TestInfoFromWS_by_msisdn", "MSISDN", "380930164453"},
		{"TestInfoFromWS_by_agreement", "AGREEMENT_CODE", "300070023"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			products, err := wsGetter.omws.Products(ctx, omws.BuildRequest(omws.P_GetProductsActContract, "IUI", omws.E{tt.key, tt.val}), true)
			assert.NoError(t, err)
			if assert.NotNil(t, products) {
				t.Logf(products.String())
				if len(products.Items) > 0 {
					t.Logf("Products=>%v", products.Find(UseCommonMain))
					t.Logf("Products_is_ACT => %t", products.Find(UseCommonMain).IsAct())
				}
			}
		})
	}

}

func Test_Acc(t *testing.T) {
	res, err := cimws.RESTCallGet[cimws.CustomerAccount](wsGetter.cimws, ctx, cimws.P_GetDataLight, "WEB", ws.E{Key: ws.MSISDN, Value: "380930164453"})
	assert.NoError(t, err)
	assert.True(t, res.IsSuccessfulAndNotEmpty())

	e := res.GetEntity()
	specAccLight := e.CustomerBilling.CustomerBillingDescBy.Get("CustomerSubscriberAccountLightSpec")
	specSubsbAccRole := e.CustomerAccountInteractions[0].BusinessIntRoleDescrBySpec.Get("SubscriberAccountRoleSpec")
	acc := &Account{
		BillingID:     e.ID,
		Status:        e.AccountStatus,
		Tariff:        okValue(ws.GetValueGeneric[string](specAccLight, "tariffName")),
		Msisdn:        okValue(ws.GetValueGeneric[string](specAccLight, "msisdn")),
		ContractNo:    okValue(ws.GetValueGeneric[string](specAccLight, "contractNo")),
		UseCommonMain: okValue(ws.GetValueGeneric[bool](specSubsbAccRole, "specSubsbAccRole")),
	}

	res, err = cimws.RESTCallGet[cimws.CustomerAccount](wsGetter.cimws, ctx, cimws.P_GetAccFull, "WEB", ws.E{Key: ws.AccID, Value: acc.BillingID})
	assert.NoError(t, err)

	e = res.GetEntity()
	balances := e.CustomerBilling.Balances
	t.Log("balances=>", utils.JsonPrettyStr(balances))
	t.Logf("Line_Main=%+v; Line_SpendingLimit=%+v; Line_CM_Usage=%+v", accBalVal(e, "Line_Main"), accBalVal(e, "Line_SpendingLimit"), accBalVal(e, "Line_CM_Usage"))
	t.Logf("IS_bal_nil=> %t", accBalVal(e, "Line_SpendingLimit") == nil)

	acc.populateBalances(e)

	t.Log("ACC_INFO=>", utils.JsonPrettyStr(acc))
}

func Test_1(t *testing.T) {
	t.Log("RESULT:  ===.")
	t.Log(wsGetter.GetContract(ctx, conNo1).String(true))

	sb := strings.Builder{}

	nowGen := time.Now()
	for v := range 10 {
		t.Log(v)
		now := time.Now()
		wsGetter.GetContract(ctx, conNo1)
		t.Logf("\n\nELAPSED: %v ms\n\n", time.Since(now).Milliseconds())
		sb.WriteString(fmt.Sprintf("loop=%d elapsed: %v ms\n", v, time.Since(now).Milliseconds()))
	}
	t.Logf("\n\nELAPSED_TOTAL: %v ms\n\n", time.Since(nowGen).Milliseconds())
	t.Log(sb.String())
}

func Test_2(t *testing.T) {
	sb := strings.Builder{}

	nowGen := time.Now()
	for v := range 1 {
		t.Log(v)
		now := time.Now()
		_, _ = wsGetter.GetAccount(utils.GenerateCtxWithRid(), "380930164453")
		t.Logf("\n\nELAPSED: %v ms\n\n", time.Since(now).Milliseconds())
		sb.WriteString(fmt.Sprintf("loop=%d elapsed: %v ms\n", v, time.Since(now).Milliseconds()))
	}
	t.Logf("\n\nELAPSED_TOTAL: %v ms\n\n", time.Since(nowGen).Milliseconds())
	t.Log(sb.String())
}

func Test_GetAgrInfo(t *testing.T) {
	contract := &Contract{
		No:       conNo1,
		Managers: map[string]interface{}{},
	}
	t.Run("Test_GetAgrInfo", func(t *testing.T) {
		// res, err := cimws.RESTCallGet[cimws.Agreement](wsGetter.cimws, ctx, cimws.P_GetAgrShort, "A", ws.E{Key: ws.ConNo, Value: conNo1})
		res, err := cimws.RESTCallGet[cimws.Agreement](wsGetter.cimws, ctx, "WEB_NEW_2002", "A", ws.E{Key: ws.ConNo, Value: conNo1})
		assert_response(t, res, err)
		os.WriteFile("WEB_NEW_2002.json", utils.Json(res.GetEntity()), 0600)
		contract.populateBalancesContract(res.GetEntity().GetAgreementOwner().GetAccFromAgreementOwner())
		contract.BillingID = ifNotNil(getAgreementOwnerNoCheck(res.GetEntity()), func(owner *cimws.BusinessInteractionRole) string { return owner.PartyRole.ID })
	})

	t.Run("Test_GetContactPersonsContract", func(t *testing.T) {
		res, err := cimws.RESTCallGet[cimws.Agreement](wsGetter.cimws, ctx, cimws.P_GetContactPersonsContract, "A", ws.E{Key: ws.ConNo, Value: conNo1})
		assert_response(t, res, err)
		contract.ContactPersons = NewContactPersons(res.GetEntity())
	})

	t.Run("Test_GetProudctAvailable", func(t *testing.T) {
		r, e := omws.RESTCallGet[omws.ProductOffering](wsGetter.omws, ctx, omws.P_GetProudctAvailable, "WEB", ws.E{ws.AgrCode, conNo1})
		assert_response(t, r, e)
		t.Log("Konsultant:", omws.FindProduct(r.Items, Konsultant).GetExtraFieldVal())
		t.Log("SalesExpert:", omws.FindProduct(r.Items, SalesExpert).GetExtraFieldVal())
		contract.Managers[Konsultant] = omws.FindProduct(r.Items, Konsultant).GetExtraFieldVal()
		contract.Managers[SalesExpert] = omws.FindProduct(r.Items, SalesExpert).GetExtraFieldVal()
	})

	t.Run("Test_GetProudctAvailable", func(t *testing.T) {
		r, e := omws.RESTCallGet1[omws.ProductOffering](wsGetter.omws, ctx, ws.BuildRequestOmExt(omws.P_GetProudctAvailable, "WEB", "", []string{Konsultant, SalesExpert}, ws.E{ws.AgrCode, conNo1}))
		assert_response(t, r, e)
		t.Log("Konsultant:", omws.FindProduct(r.Items, Konsultant).GetExtraFieldVal())
		t.Log("SalesExpert:", omws.FindProduct(r.Items, SalesExpert).GetExtraFieldVal())
		contract.Managers[Konsultant] = omws.FindProduct(r.Items, Konsultant).GetExtraFieldVal()
		contract.Managers[SalesExpert] = omws.FindProduct(r.Items, SalesExpert).GetExtraFieldVal()
	})

	t.Log(utils.JsonPrettyStr(contract))
}

func Test_OperateCustomerAccountsRESTCall_3(t *testing.T) {
	t.Log(wsGetter.GetFMC(ctx, "300070023", "380930164453"))
}

func Test_OperateCustomerAccountsRESTCall_2(t *testing.T) {
	processResp := func(t *testing.T, settings string) {
		t.Log(">>> SETTINGS: ", settings)
		var response WSResponse
		err := json.Unmarshal([]byte(settings), &response)
		if err != nil {
			t.Logf("Error unmarshaling JSON: %v", err)
		}

		if len(response) == 0 {
			t.Log("No response")
			return
		}
		printFMCResponse(response)

		phoneData1 := aggregatePhoneInfo(response, "380930164453")
		t.Logf("=================\n%s\n=================\n", phoneData1)
	}

	t.Run("First", func(t *testing.T) {
		result, err := cimws.RESTCallOperate(wsGetter.cimws, ctx, cimws.CreateRequestOperateAcc(FMC_VOIP, "Iguana", "IUI", cimws.KV{"contractCode": "300070023", "taxSchemaCode": "taxSchemaCode", "creditLimit": "1000"}))
		assert.NoError(t, err)
		assert.NotNil(t, result.ChainResults)
		t.Log(ws.StringObj(result, true))
	})

	t.Run("call-FMC_GET_VOIP", func(t *testing.T) {
		result, err := cimws.RESTCallOperate(wsGetter.cimws, ctx, cimws.CreateRequestOperateAcc(FMC_VOIP, "Iguana", "IUI", cimws.KV{"contractCode": "300070023", "taxSchemaCode": "taxSchemaCode", "creditLimit": "1000"}))
		assert.NoError(t, err)
		assert.NotNil(t, result.ChainResults)
		t.Log(ws.StringObj(result, true))

		settings := result.ChainResults[0].GetVarValue("settings")
		processResp(t, settings)
	})

	t.Run("call-GET_FMC_MOBILE", func(t *testing.T) {
		result, err := cimws.RESTCallOperate(wsGetter.cimws, ctx, cimws.CreateRequestOperateAcc(FMC_MOBILE, "Iguana", "IUI", cimws.KV{"contractCode": "300070023"}))
		assert.NoError(t, err)
		assert.NotNil(t, result.ChainResults)
		t.Log(ws.StringObj(result, true))

		if len(result.ChainResults) == 0 {
			t.Log("No chain results")
			return
		}

		settings := result.ChainResults[0].GetVarValue("settings")
		processResp(t, settings)
	})

	t.Run("call-GET_FMC_PREFIX", func(t *testing.T) {
		result, err := cimws.RESTCallOperate(wsGetter.cimws, ctx, cimws.CreateRequestOperateAcc(FMC_PREFIX, "Iguana", "IUI", cimws.KV{"contractCode": "300070023"}))
		assert.NoError(t, err)
		assert.NotNil(t, result.ChainResults)
		t.Log(ws.StringObj(result, true))

		if len(result.ChainResults) == 0 {
			t.Log("No chain results")
			return
		}

		settings := result.ChainResults[0].GetVarValue("settings")
		processResp(t, settings)
	})

	t.Run("Third", func(t *testing.T) {
		// GetFMC
		phoneData, err := wsGetter.GetFMC(ctx, "300070023", "380930164453")
		assert.NoError(t, err)
		t.Log(phoneData)
	})

}
func Help_table_setup(t *testing.T, tw *tablewriter.Table, cols []string) {
	t.Helper()
	tw.SetHeader(cols)
	tw.SetAlignment(tablewriter.ALIGN_LEFT)
	tw.SetColWidth(75)
	tw.SetAutoWrapText(false)
	// tw.SetTablePadding("\t\t--->")
	tw.SetRowSeparator("*")
	// tw.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	// tw.SetCenterSeparator("|")
	tw.SetColumnColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlackColor},
	)
	t.Log("Table was set up!")
}
func Test_get_fmc_profiles(t *testing.T) {
	dict := map[string]int{
		"VOIP":   0,
		"MOBILE": 1,
		"PREFIX": 2,
	}
	row1 := make([]string, 3)
	row2 := make([]string, 3)
	tw := tablewriter.NewWriter(os.Stdout)
	Help_table_setup(t, tw, []string{"VOIP", "MOBILE", "PREFIX"})

	processResp := func(t *testing.T, resp *ws.OperateResponseREST) {
		settings := get_settings(resp, t)
		if settings == "" {
			return
		}
		t.Logf(">>> SETTINGS[%s]: %s\n", t.Name(), settings)
		row1[dict[strings.Split(t.Name(), "/")[1]]] = settings
		var response WSResponse
		err := json.Unmarshal([]byte(settings), &response)
		if err != nil {
			t.Logf("Error unmarshaling JSON: %v", err)
		}

		if len(response) == 0 {
			t.Log("No response")
			return
		}

		b, _ := json.MarshalIndent(response, "", "  ")
		row2[dict[strings.Split(t.Name(), "/")[1]]] = string(b)
		printFMCResponse(response)

		phoneData1 := aggregatePhoneInfo(response, "380930164453")
		t.Logf("=================\n%s\n=================\n", phoneData1)
	}

	conNo := "300070023"

	t.Run("VOIP", func(t *testing.T) {
		result, err := cimws.RESTCallOperate(wsGetter.cimws, ctx, cimws.CreateRequestOperateAcc(FMC_VOIP, "Iguana", "IUI", cimws.KV{"contractCode": conNo}))
		assert.NoError(t, err)
		assert.NotNil(t, result.ChainResults)
		t.Log(ws.StringObj(result, true))
		processResp(t, result)
	})

	t.Run("MOBILE", func(t *testing.T) {
		result, err := cimws.RESTCallOperate(wsGetter.cimws, ctx, cimws.CreateRequestOperateAcc(FMC_MOBILE, "Iguana", "IUI", cimws.KV{"contractCode": conNo}))
		assert.NoError(t, err)
		assert.NotNil(t, result.ChainResults)
		t.Log(ws.StringObj(result, true))
		processResp(t, result)
	})

	t.Run("PREFIX", func(t *testing.T) {
		t.Log("Dg")
		result := get_fmc_t(t, FMC_PREFIX, conNo)
		assert.NotNil(t, result.ChainResults)
		t.Log(ws.StringObj(result, true))
		processResp(t, result)

	})

	tw.Append(row1)
	tw.Append(row2)
	tw.Render()

}

func get_settings(resp *ws.OperateResponseREST, t *testing.T) string {
	if len(resp.ChainResults) == 0 || len(resp.ChainResults[0].Result) == 0 {
		t.Log("No chain results")
		return ""
	}
	settings := resp.ChainResults[0].GetVarValue("settings")
	return settings
}

func get_fmc_t(t *testing.T, p, conNo string) *ws.OperateResponseREST {
	t.Helper()
	result, err := cimws.RESTCallOperate(wsGetter.cimws, ctx, cimws.CreateRequestOperateAcc(p, "Iguana", "IUI", cimws.KV{"contractCode": conNo}))
	if !assert.NoError(t, err) {
		t.Fatalf("Error calling RESTCallOperate: %v", err)
	}
	if len(result.ChainResults) == 0 || len(result.ChainResults[0].Result) == 0 {
		t.Fatal("No chain results")
	}
	return result
}

func TestWSGetter_getFMC_mobile(t *testing.T) {
	tw := tablewriter.NewWriter(os.Stdout)
	tw.SetAutoWrapText(false)
	tw.SetColWidth(115)
	tw.SetAlignment(tablewriter.ALIGN_LEFT)

	t.Run("GetFMC_VOIP_for_contract", func(t *testing.T) {
		t.Log("---")
		// for _, v := range []struct{ conNo, msisdn string }{{"300070023", ""}, {"300070023", "380933780687"}} {
		for _, v := range []struct{ conNo, msisdn string }{
			{"300070023", ""},
			{"300070023", "380930164453"},
			{"300070023","380933780687"},
		} {
			r, e := wsGetter.GetFMC_VOIP(ctx, v.conNo, v.msisdn)
			if !assert.NoError(t, e) {
				t.Fatal(e)
			}
			t.Log(utils.JsonPrettyStr(r))
			settingsoriginal := get_fmc_t(t, FMC_VOIP, v.conNo).ChainResults[0].GetVarValue("settings")
			tw.Append([]string{fmt.Sprintf("contract=%s\nMSISDN=%s", v.conNo, v.msisdn), settingsoriginal, jsonPretty(r)})
		}

	})

	t.Run("GetFMC_Mobile_for_contract", func(t *testing.T) {
		r, e := wsGetter.GetFMC_MOBILE(ctx, conNo1, "")
		if e != nil {
			t.Fatal(e)
		}
		t.Log(utils.JsonPrettyStr(r))
	})

	t.Run("GetFMC_Mobile_for_msisdn", func(t *testing.T) {
		t.Log("s")
		r, e := wsGetter.GetFMC_MOBILE(ctx, conNo1, "380933780678")
		if e != nil {
			t.Fatal(e)
		}
		t.Log(utils.JsonPrettyStr(r))
		settingsoriginal := get_fmc_t(t, FMC_PREFIX, conNo1).ChainResults[0].GetVarValue("settings")
		settingsMobile := get_fmc_t(t, FMC_MOBILE, conNo1).ChainResults[0].GetVarValue("settings")
		tw.Append([]string{settingsMobile, settingsoriginal, jsonPretty(r)})
	})
	tw.Render()

}

func TestWSGetter_getFMC(t *testing.T) {
	type args struct {
		log        *gologgers.LogRec
		profile    string
		contractNo string
		msisdn     string
	}
	tests := []struct {
		name    string
		args    args
		want    *AggregatedPhoneInfo
		wantErr bool
	}{
		{
			name: "Successful FMC retrieval",
			args: args{
				log:        l.RecWithCtx(ctx, ch),
				profile:    FMC_VOIP,
				contractNo: "300070023",
				msisdn:     "380930164453",
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "FMC retrieval with error",
			args: args{
				log:        l.RecWithCtx(ctx, ch),
				profile:    FMC_VOIP,
				contractNo: "300070023",
				msisdn:     "380930164400",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Empty FMC response",
			args: args{
				log:        l.RecWithCtx(ctx, ch),
				profile:    FMC_VOIP,
				contractNo: "300070023",
				msisdn:     "380930164453",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := wsGetter.getFMC(tt.args.log, tt.args.profile, tt.args.contractNo, tt.args.msisdn)
			if (err != nil) != tt.wantErr {
				t.Errorf("WSGetter.getFMC() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Log(got)
		})
	}
}

func assert_response[T any](t *testing.T, res *ws.GetResponseREST[T], err error) {
	t.Helper()
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	if !assert.True(t, res.IsSuccessfulAndNotEmpty()) {
		t.Log(res.String(true))
	}
}

func jsonPretty(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
