package main

import (
	"flag"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"k8s.io/klog/v2"

	dynupdate "github.com/akyriako/aws-cgw-dynupdate"
)

var (
	vpnConnectionId = flag.String("vpnConnectionId", "", "VPN Connection ID")
	sess            *session.Session
)

func main() {
	defer exit()

	klog.Infof("Started dynamic IP address update for VPN: %s", *vpnConnectionId)

	//svc := ec2.New(sess, aws.NewConfig().WithLogLevel(aws.LogDebug))
	svc := ec2.New(sess, aws.NewConfig())

	err := dynupdate.UpdateCgwDynamicIpAddress(svc, *vpnConnectionId)
	if err != nil {
		klog.Fatalln(err)
	}
}

func exit() {
	exitCode := 10
	klog.Infof("Finished dynamic IP address update for VPN: %s", *vpnConnectionId)
	klog.FlushAndExit(klog.ExitFlushTimeout, exitCode)
}

func init() {
	klog.InitFlags(nil)
	flag.Parse()

	var err error

	sess, err = initSession()
	if err != nil {
		klog.Fatalln(err)
	}
}

func initSession() (sess *session.Session, err error) {
	// Initialize a session, credentials will be loaded from ~/.aws/credentials
	sess, err = session.NewSessionWithOptions(session.Options{
		// Force enable Shared Config support
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return nil, err
	}

	_, err = sess.Config.Credentials.Get()
	if err != nil {
		return nil, err
	}

	return sess, nil
}
