module antrea.io/resource-auditing

go 1.16

replace (
	github.com/Microsoft/hcsshim v0.8.9 => github.com/ruicao93/hcsshim v0.8.10-0.20210114035434-63fe00c1b9aa
	github.com/contiv/ofnet => github.com/wenyingd/ofnet v0.0.0-20210318032909-171b6795a2da
)

require (
	antrea.io/antrea v1.2.0
	github.com/go-git/go-billy/v5 v5.3.1
	github.com/go-git/go-git/v5 v5.4.2
	k8s.io/client-go v0.21.3
	k8s.io/klog/v2 v2.10.0
)
