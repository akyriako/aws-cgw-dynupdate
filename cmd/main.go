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
	vpnConnectionId = flag.String("vpnConnectionId", "", "vpn connection id")
	debug           = flag.Bool("debug", false, "debug mode")
	maxRetries      = flag.Uint("maxRetries", 3, "max retries on request failure")
	sess            *session.Session
	svc             *ec2.EC2
)

func main() {
	defer exit()

	klog.Infof("Started dynamic IP address update for VPN: %s", *vpnConnectionId)

	if *debug {
		svc = ec2.New(sess, aws.NewConfig().WithLogLevel(aws.LogDebug).WithMaxRetries(int(*maxRetries)))
	} else {
		svc = ec2.New(sess, aws.NewConfig().WithMaxRetries(int(*maxRetries)))
	}

	err := dynupdate.UpdateCgwDynamicIpAddress(svc, vpnConnectionId)
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
