module github.com/containersol/prescale-operator

go 1.15

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-logr/logr v0.3.0
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/olekukonko/tablewriter v0.0.0-20170122224234-a0225b3f23b5
	github.com/onsi/ginkgo v1.15.2
	github.com/onsi/gomega v1.10.2
	github.com/openshift/api v3.9.0+incompatible
	github.com/prometheus/common v0.10.0
	github.com/robfig/cron v1.2.0
	golang.org/x/sys v0.0.0-20210320140829-1e4c9ba3b0c4 // indirect
	golang.org/x/tools v0.1.1-0.20210222172741-77e031214674 // indirect
	honnef.co/go/tools v0.1.1 // indirect
	k8s.io/api v0.20.5
	k8s.io/apimachinery v0.20.5
	k8s.io/client-go v0.20.4
	k8s.io/kubectl v0.20.4
	sigs.k8s.io/controller-runtime v0.7.0
)
