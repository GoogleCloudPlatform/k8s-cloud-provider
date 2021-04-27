module github.com/GoogleCloudPlatform/k8s-cloud-provider

go 1.13

require (
	golang.org/x/oauth2 v0.0.0-20210413134643-5e61552d6c78
	google.golang.org/api v0.45.0
	k8s.io/klog/v2 v2.0.0
)

replace (
	cloud.google.com/go => cloud.google.com/go v0.51.0
	golang.org/x/net => golang.org/x/net v0.0.0-20200114155413-6afb5195e5aa
	golang.org/x/oauth2 => golang.org/x/oauth2 v0.0.0-20191202225959-858c2ad4c8b6
	golang.org/x/sys => golang.org/x/sys v0.0.0-20200116001909-b77594299b42
	google.golang.org/api => google.golang.org/api v0.45.0
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20200115191322-ca5a22157cba
)
