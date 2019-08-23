package main

import (
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/lib/cid"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func authenticateSupplier(stub shim.ChaincodeStubInterface) error {
	mspID, issuerCN, _ := getCreatorInfo(stub)
	if mspID == "SupplierMSP" && issuerCN == "ca.supplier.scmn.com" {
		return nil
	}
	return createAuthenticationFailedError("SupplierMSP", mspID)
}

func authenticateProducer(stub shim.ChaincodeStubInterface) error {
	mspID, issuerCN, _ := getCreatorInfo(stub)
	if mspID == "ProducerMSP" && issuerCN == "ca.producer.scmn.com" {
		return nil
	}
	return createAuthenticationFailedError("ProducerMSP", mspID)
}

func authenticateDistributor(stub shim.ChaincodeStubInterface) error {
	mspID, issuerCN, _ := getCreatorInfo(stub)
	if mspID == "DistributorMSP" && issuerCN == "ca.distributor.scmn.com" {
		return nil
	}
	return createAuthenticationFailedError("DistributorMSP", mspID)
}

func authenticateCustomer(stub shim.ChaincodeStubInterface) error {
	mspID, issuerCN, _ := getCreatorInfo(stub)
	if mspID == "CustomerMSP" && issuerCN == "ca.customer.scmn.com" {
		return nil
	}
	return createAuthenticationFailedError("CustomerMSP", mspID)
}

func getCreatorInfo(stub shim.ChaincodeStubInterface) (string, string, error) {
	var mspid string
	var cert *x509.Certificate
	var err error

	mspid, err = cid.GetMSPID(stub)
	if err != nil {
		fmt.Printf("Error getting MSP identity")
		return "", "", err
	}

	cert, err = cid.GetX509Certificate(stub)
	if err != nil {
		fmt.Printf("Error getting X509 certificate")
		return "", "", err
	}

	return mspid, cert.Issuer.CommonName, nil
}

func createAuthenticationFailedError(expectedMspID string, actualMspID string) error {
	return errors.New(fmt.Sprintf("Authenticaiton failed: Expected '%t' but was '%t'", expectedMspID, actualMspID))
}
