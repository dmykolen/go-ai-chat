package biz

import (
	"github.com/samber/lo"
	"gitlab.dev.ict/golang/libs/ws/cimws"
)

func okValue[T any](v T, ok bool) T {
	return v
}

func ifOk[T any, R any](vv T, ok bool, xx func(val T) R) (res R) {
	if !ok {
		return
	}
	return xx(vv)
}

func ifNotNil[T any, R any](vv *T, xx func(val *T) R) (res R) {
	if vv == nil {
		return
	}
	return xx(vv)
}

func accBalVal(e *cimws.CustomerAccount, name string) *float64 {
	b := e.CustomerBilling.FindBalanceByCode(name)
	return bVal(b)
}

func bVal(b *cimws.Balance) *float64 {
	if b == nil {
		return nil
	}
	return &b.Value
}

func getAgreementOwner(e *cimws.Agreement) (owner *cimws.BusinessInteractionRole, isFound bool) {
	agrOwner, ok := lo.Find[cimws.BusinessInteractionRole](e.BusinessInteractionRoles, func(item cimws.BusinessInteractionRole) bool {
		return item.Class == "ukr.astelit.cim.model.customeracc.role.AgreementOwner"
	})
	if !ok {
		return nil, ok
	}
	return &agrOwner, ok
}

func getAgreementOwnerNoCheck(e *cimws.Agreement) *cimws.BusinessInteractionRole {
	return okValue(getAgreementOwner(e))
}

func getAccFromAgreementOwner(owner *cimws.BusinessInteractionRole) *cimws.CustomerAccount {
	if owner == nil || owner.PartyRole == nil || len(owner.PartyRole.CustomerPossess) == 0 {
		return nil
	}
	return &owner.PartyRole.CustomerPossess[0]
}

func GetAgrAccId(a *cimws.Agreement) (accId string) {
	if a == nil {
		return
	}

	if owner := a.GetAgreementOwner(); owner != nil && owner.PartyRole != nil {
		return owner.PartyRole.ID
	}
	return
}
