package main

import (
	"github.com/google/uuid"
	"time"
)

type ProductOrder struct {
	OrderId                uuid.UUID          `json:"orderId"`
	ProductId              uuid.UUID          `json:"productId"`
	ProductPartsTotalCount int                `json:"productPartsTotalCount"`
	ProductPartOrders      []ProductPartOrder `json:"productPartOrders"`
	OrderedTimestamp       time.Time          `json:"orderedTimestamp"`
	ManufacturedTimestamp  time.Time          `json:"manufacturedTimestamp"`
	DeliveredTimestamp     time.Time          `json:"deliveredTimestamp"`
	State                  string             `json:"state"`
	MspIdOfRejecter        *string            `json:"mspIdOfRejecter"`
}

type ProductOrderState int

const (
	ORDERED ProductOrderState = iota
	ACCEPTED
	MANUFACTURED
	DELIVERED
	REJECTED
)

var VALID_PRODUCT_ORDER_STATES = [...]string{"ORDERED", "ACCEPTED", "MANUFACTURED", "DELIVERED", "REJECTED"}

func (i ProductOrderState) String() string {
	return VALID_PRODUCT_ORDER_STATES[i]
}

type ProductPartOrder struct {
	OrderId            uuid.UUID `json:"orderId"`
	ProductPartId      uuid.UUID `json:"productPartId"`
	OrderedTimestamp   time.Time `json:"orderedTimestamp"`
	DeliveredTimestamp time.Time `json:"deliveredTimestamp"`
	State              string    `json:"state"`
}

type ProductPartOrderState int

const (
	PART_ORDERED ProductPartOrderState = iota
	PART_DELIVERED
)

var VALID_PRODUCT_PART_ORDER_STATES = [...]string{"ORDERED", "DELIVERED"}

func (i ProductPartOrderState) String() string {
	return VALID_PRODUCT_PART_ORDER_STATES[i]
}

func productOrderStateIsInvalid(code string) bool {
	for _, validCode := range VALID_PRODUCT_ORDER_STATES {
		if validCode == code {
			return false
		}
	}
	return true
}

func productPartOrderStateIsInvalid(code string) bool {
	for _, validCode := range VALID_PRODUCT_PART_ORDER_STATES {
		if validCode == code {
			return false
		}
	}
	return true
}
