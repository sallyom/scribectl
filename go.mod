module github.com/backube/scribectl

go 1.15

require (
	github.com/backube/scribe v0.1.0
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/cli-runtime v0.20.4
	k8s.io/client-go v0.20.4
	k8s.io/component-base v0.20.4
	k8s.io/klog/v2 v2.4.0
	k8s.io/kubectl v0.20.4
	sigs.k8s.io/controller-runtime v0.6.2
)

replace github.com/backube/scribectl => /home/somalley/code/gowork/src/github.com/backube/scribectl
