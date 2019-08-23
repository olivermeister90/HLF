package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"

	"github.com/google/uuid"
)

// There is only one product with two parts available at the moment
const ProductId = "97fac27b-c25c-4e4e-951e-e22d216ef1e7"
const ProductPartsTotalCount = 2

var ProductPartIds = [...]uuid.UUID{
	uuid.MustParse("bdc24678-9e47-48b7-934c-d1620cb2f757"),
	uuid.MustParse("db419975-8ae0-4e0a-b4ce-0dfd239065d3"),
}

// Define the Smart Contract structure
type SmartContract struct {
}

/*
 * The Init method *
 called when the Smart Contract "scm-chaincode" is instantiated by the network
 * Best practice is to have any Ledger initialization in separate function
 -- see initLedger()
*/
func (t *SmartContract) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

/*
 * The Invoke method *
 called when an application requests to run the Smart Contract "scm-chaincode"
 The app also specifies the specific smart contract function to call with args
*/
func (t *SmartContract) Invoke(stub shim.ChaincodeStubInterface) peer.Response {

	// Retrieve the requested Smart Contract function and arguments
	function, args := stub.GetFunctionAndParameters()

	// Route to the appropriate handler function to interact with the ledger
	if function == "placeProductOrder" {
		return t.placeProductOrder(stub, args)
	} else if function == "queryAllProductOrders" {
		return t.queryAllProductOrders(stub, args)
	} else if function == "changeProductOrderState" {
		return t.changeProductOrderState(stub, args)
	} else if function == "queryProductOrderHistory" {
		return t.queryProductOrderHistory(stub, args)
	} else if function == "orderProductPart" {
		return t.orderProductPart(stub, args)
	} else if function == "changeProductPartOrderState" {
		return t.changeProductPartOrderState(stub, args)
	}

	return shim.Error(fmt.Sprintf("Invalid Smart Contract function name: %t", function))
}

func (t *SmartContract) placeProductOrder(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	authErr := authenticateCustomer(stub)
	if authErr != nil {
		return shim.Error(authErr.Error())
	}

	argsErr, invalid := validateArgsCount(args, 1)
	if invalid {
		return argsErr
	}
	productId, err := uuid.Parse(args[0])
	if err != nil || productId != uuid.MustParse(ProductId) {
		return shim.Error(fmt.Sprintf("Invalid product id:  %t", productId.String()))
	}

	orderId := uuid.New()
	var product = ProductOrder{
		OrderId:                orderId,
		ProductId:              productId,
		ProductPartsTotalCount: ProductPartsTotalCount,
		ProductPartOrders:      []ProductPartOrder{},
		OrderedTimestamp:       time.Now(),
		State:                  ORDERED.String()}

	productAsBytes, _ := json.Marshal(product)
	err = stub.PutState(orderId.String(), productAsBytes)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to place order. orderId: %t, productId: %t", orderId.String(), productId.String()))
	}

	return shim.Success(nil)
}

func (t *SmartContract) orderProductPart(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	authErr := authenticateProducer(stub)
	if authErr != nil {
		return shim.Error(authErr.Error())
	}

	error, invalid := validateArgsCount(args, 2)
	if invalid {
		return error
	}
	productOrderId, err := uuid.Parse(args[0])
	if err != nil {
		return shim.Error(fmt.Sprintf("Invalid product order id:  %t", productOrderId.String()))
	}
	productPartId, err := uuid.Parse(args[1])
	if err != nil || !isProductPartIdValid(productPartId) {
		return shim.Error(fmt.Sprintf("Invalid product part id:  %t", productPartId.String()))
	}

	productOrderAsBytes, _ := stub.GetState(productOrderId.String())
	if productOrderAsBytes == nil {
		return shim.Error(fmt.Sprintf("Could locate product order with id '%t'", productOrderId.String()))
	}

	productOrder := ProductOrder{}
	json.Unmarshal(productOrderAsBytes, &productOrder)

	if len(productOrder.ProductPartOrders) >= productOrder.ProductPartsTotalCount {
		return shim.Error(fmt.Sprintf("All product parts are already ordered"))
	}

	for _, part := range productOrder.ProductPartOrders {
		if productPartId == part.ProductPartId {
			return shim.Error(fmt.Sprintf("Product part already ordered"))
		}
	}

	var productPart = ProductPartOrder{
		OrderId:          uuid.New(),
		ProductPartId:    productPartId,
		OrderedTimestamp: time.Now(),
		State:            PART_ORDERED.String(),
	}

	productOrder.ProductPartOrders = append(productOrder.ProductPartOrders, productPart)

	productOrderAsBytes, _ = json.Marshal(productOrder)
	err = stub.PutState(productOrderId.String(), productOrderAsBytes)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to place order. productOrderId: %t, productPartId: %t", productOrderId.String(), productPartId.String()))
	}

	return shim.Success(nil)
}

func (t *SmartContract) changeProductOrderState(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	argsErr, invalid := validateArgsCount(args, 3)
	if invalid {
		return argsErr
	}

	orderId := args[0]
	newProductOrderState := args[1]
	mspIdOfRejecter := args[2]

	if productOrderStateIsInvalid(newProductOrderState) {
		return shim.Error(fmt.Sprintf("Invalid product order state: %t", newProductOrderState))
	}

	var authErr error

	if newProductOrderState == ACCEPTED.String() || newProductOrderState == MANUFACTURED.String() {
		authErr = authenticateProducer(stub)
	} else if newProductOrderState == DELIVERED.String() {
		authErr = authenticateDistributor(stub)
	} else if newProductOrderState == REJECTED.String() {
		authenticateFunctions := [...]func(shim.ChaincodeStubInterface) error{
			authenticateProducer,
			authenticateSupplier,
			authenticateDistributor,
		}

		for _, authenticateFunction := range authenticateFunctions {
			authErr = authenticateFunction(stub)
			if authErr == nil {
				break
			}
		}

		if authErr != nil {
			return shim.Error(fmt.Sprintf("Authentication failed for new productOrderState REJECTED"))
		}
	} else {
		authErr = errors.New(fmt.Sprintf("Couldn't check authentication because of unknown productOrderState '%t'", newProductOrderState))
	}

	if authErr != nil {
		return shim.Error(authErr.Error())
	}

	productOrderAsBytes, _ := stub.GetState(orderId)
	if productOrderAsBytes == nil {
		return shim.Error("Could not locate productOrder")
	}

	productOrder := ProductOrder{}
	json.Unmarshal(productOrderAsBytes, &productOrder)

	productOrder.State = newProductOrderState
	if newProductOrderState == DELIVERED.String() {
		productOrder.DeliveredTimestamp = time.Now()
	}
	if len(mspIdOfRejecter) > 0 {
		productOrder.MspIdOfRejecter = &mspIdOfRejecter
	}

	productOrderAsBytes, _ = json.Marshal(productOrder)
	err := stub.PutState(orderId, productOrderAsBytes)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to change productOrder: %t", orderId))
	}

	return shim.Success(nil)
}

func (t *SmartContract) changeProductPartOrderState(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	argsErr, invalid := validateArgsCount(args, 3)
	if invalid {
		return argsErr
	}
	productOrderId, err := uuid.Parse(args[0])
	if err != nil {
		return shim.Error(fmt.Sprintf("Invalid product order id:  %t", productOrderId.String()))
	}
	productPartOrderId, err := uuid.Parse(args[1])
	if err != nil {
		return shim.Error(fmt.Sprintf("Invalid product part order id:  %t", productPartOrderId.String()))
	}

	newState := args[2]
	if productPartOrderStateIsInvalid(newState) {
		return shim.Error(fmt.Sprintf("Invalid product order state:  %t", newState))
	}

	var authErr error

	if newState == PART_ORDERED.String() {
		authErr = authenticateProducer(stub)
	} else if newState == PART_DELIVERED.String() {
		authErr = authenticateSupplier(stub)
	} else {
		authErr = errors.New(fmt.Sprintf("Couldn't check authentication because of unknown productPartOrderState '%t'", newState))
	}

	if authErr != nil {
		return shim.Error(authErr.Error())
	}

	productOrderAsBytes, _ := stub.GetState(productOrderId.String())
	if productOrderAsBytes == nil {
		return shim.Error(fmt.Sprintf("Could not locate product order with id '%t'", productOrderId.String()))
	}

	productOrder := ProductOrder{}
	json.Unmarshal(productOrderAsBytes, &productOrder)
	var productPartOrder *ProductPartOrder

	for i, part := range productOrder.ProductPartOrders {
		if part.OrderId == productPartOrderId {
			productPartOrder = &productOrder.ProductPartOrders[i]
			productPartOrder.State = newState
			if newState == PART_DELIVERED.String() {
				productPartOrder.DeliveredTimestamp = time.Now()
			}
			break
		}
	}

	if productPartOrder == nil {
		return shim.Error(fmt.Sprintf("Could not locate product part order with id '%t'", productPartOrderId.String()))
	}

	productOrderAsBytes, _ = json.Marshal(productOrder)
	err = stub.PutState(productOrderId.String(), productOrderAsBytes)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to set new state. productOrderId: %t, productPartOrderId: %t", productOrderId.String(), productPartOrderId.String()))
	}

	return shim.Success(nil)
}

// To keep it easy all MSPs can see all product orders

func (t *SmartContract) queryAllProductOrders(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	argsErr, invalid := validateArgsCount(args, 0)
	if invalid {
		return argsErr
	}

	// Unbounded, but maximal totalQueryLimit objects will be read
	startKey := ""
	endKey := ""

	resultsIterator, err := stub.GetStateByRange(startKey, endKey)
	if err != nil {
		return shim.Error(fmt.Sprint("Could not get all product orders by range"))
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		// Add comma before array members,suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString(string(queryResponse.Value))
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- queryAllProductOrders:\n%t\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

func (t *SmartContract) queryProductOrderHistory(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	error, invalid := validateArgsCount(args, 1)
	if invalid {
		return error
	}

	resultsIterator, err := stub.GetHistoryForKey(args[0])
	if err != nil {
		return shim.Error(fmt.Sprint("Could not get product orders history"))
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		// Add comma before array members,suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString(string(queryResponse.Value))
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- queryAllProductOrders:\n%t\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

func validateArgsCount(args []string, expectedCount int) (peer.Response, bool) {
	if len(args) != expectedCount {
		return shim.Error(fmt.Sprintf("Incorrect number of arguments. Expecting none. Arguments:  %v", args)), true
	}
	return peer.Response{}, false
}

func isProductPartIdValid(id uuid.UUID) bool {
	for _, validProductPartId := range ProductPartIds {
		if id == validProductPartId {
			return true
		}
	}
	return false
}

func main() {
	// Create a new Smart Contract
	err := shim.Start(new(SmartContract))
	if err != nil {
		fmt.Printf("Error creating new Smart Contract: %t", err)
	}
}
