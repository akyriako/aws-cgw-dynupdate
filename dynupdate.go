package aws_cgw_dynupdate

import (
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"k8s.io/klog/v2"
)

const (
	externalIpRawUrl string = "https://myexternalip.com/raw"
)

var (
	vpnConnectionId   string
	customerGatewayId string
)

func UpdateCgwDynamicIpAddress(svc *ec2.EC2, vpnId string) error {
	vpnConnectionId = vpnId

	vpnOutput, err := describeVpnConnection(svc)
	if err != nil {
		return err
	}

	customerGatewayId = *vpnOutput.VpnConnections[0].CustomerGatewayId
	cgwOutput, err := describeCustomerGateway(svc)
	if err != nil {
		return err
	}

	oldIpAddress := *cgwOutput.CustomerGateways[0].IpAddress
	newIpAddress, err := getExternalIP()
	if err != nil {
		return err
	}

	if oldIpAddress == newIpAddress {
		klog.Infof("No sync required, IP address remains: %s", oldIpAddress)
		return nil
	}

	klog.Infof("Updating CGW IP address from %s to %s...", oldIpAddress, newIpAddress)

	newCgwOutput, err := createCustomerGateway(
		svc,
		*cgwOutput.CustomerGateways[0].Type,
		newIpAddress,
		*cgwOutput.CustomerGateways[0].BgpAsn,
	)
	if err != nil {
		klog.Infof("Creating new CGW with remote IP address: %s failed!", newIpAddress)
		return err
	}

	klog.Infof("Created CGW IP with remote IP address: %s", *newCgwOutput.CustomerGateway.IpAddress)

	mvpnOutput, err := modifyVpnConnection(svc, *vpnOutput.VpnConnections[0].VpnConnectionId, *newCgwOutput.CustomerGateway.CustomerGatewayId)
	if err != nil {
		klog.Infof("Assigning CGW: %s to VPN: %s failed!", *newCgwOutput.CustomerGateway.CustomerGatewayId, *mvpnOutput.VpnConnection.VpnConnectionId)
		return err
	}

	klog.Infof("Assigned CGW: %s to VPN: %s", *newCgwOutput.CustomerGateway.CustomerGatewayId, *mvpnOutput.VpnConnection.VpnConnectionId)

	_, err = deleteCustomerGateway(svc, customerGatewayId)
	if err != nil {
		klog.Infof("Deleting old CGW with remote IP address: %s failed!", oldIpAddress)
		return err
	}

	return nil
}

func describeVpnConnection(svc *ec2.EC2) (vpnOutput *ec2.DescribeVpnConnectionsOutput, err error) {
	vpnInput := &ec2.DescribeVpnConnectionsInput{
		VpnConnectionIds: []*string{aws.String(vpnConnectionId)},
	}

	vpnOutput, err = svc.DescribeVpnConnections(vpnInput)
	if err != nil {
		return nil, err
	}

	return
}

func modifyVpnConnection(svc *ec2.EC2, vpnId, cgwId string) (vpnOutput *ec2.ModifyVpnConnectionOutput, err error) {
	vpnInput := &ec2.ModifyVpnConnectionInput{
		VpnConnectionId:   aws.String(vpnId),
		CustomerGatewayId: aws.String(cgwId),
	}

	vpnOutput, err = svc.ModifyVpnConnection(vpnInput)
	if err != nil {
		return nil, err
	}

	return vpnOutput, nil
}

func describeCustomerGateway(svc *ec2.EC2) (customerGatewayOutput *ec2.DescribeCustomerGatewaysOutput, err error) {
	customerGatewayInput := &ec2.DescribeCustomerGatewaysInput{
		CustomerGatewayIds: []*string{aws.String(customerGatewayId)},
	}

	customerGatewayOutput, err = svc.DescribeCustomerGateways(customerGatewayInput)
	if err != nil {
		return nil, err
	}

	return customerGatewayOutput, nil
}

func createCustomerGateway(svc *ec2.EC2, cgwType, ipAddress, bgpAsn string) (cgwOutput *ec2.CreateCustomerGatewayOutput, err error) {
	bgpAsnInt64, err := strconv.ParseInt(bgpAsn, 10, 64)
	if err != nil {
		return nil, err
	}

	cgwInput := &ec2.CreateCustomerGatewayInput{
		BgpAsn:   aws.Int64(bgpAsnInt64),
		PublicIp: aws.String(ipAddress),
		Type:     aws.String(cgwType),
	}

	cgwOutput, err = svc.CreateCustomerGateway(cgwInput)
	if err != nil {
		return nil, err
	}

	return cgwOutput, nil
}

func deleteCustomerGateway(svc *ec2.EC2, cgwId string) (cgwOutput *ec2.DeleteCustomerGatewayOutput, err error) {
	cgwInput := &ec2.DeleteCustomerGatewayInput{
		CustomerGatewayId: aws.String(cgwId),
	}

	cgwOutput, err = svc.DeleteCustomerGateway(cgwInput)
	if err != nil {
		return nil, err
	}

	return cgwOutput, nil
}

func getExternalIP() (string, error) {
	resp, err := http.Get(externalIpRawUrl)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}(resp.Body)

	ipaddress, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(ipaddress), nil
}
