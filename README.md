# Pre-Scaling Kubernetes Operator

Built out of necessity, the Operator helps pre-scale applications in anticipation of load.

At its core, it manages a cluster / namespace "state", 
which triggers scaling of applications to the desired replicas for that specific state. 

For example, a cluster could have 2 states, `business-as-usual`, and `peak`.
Applications can configure the replicas needed for each state. When a cluster is switched from one state to the other, the operator will take care of scaling all of the applications to the necessary replicas, as set by developers through annotations.

From that point forward, the operator is controlling the size of all the applications that have chosen to opt-in. This is possible through a label. We ensure in that way that the state of the cluster is maintained. The operator can allow applications to autoscale through a specific annotation.


## Use cases

This operator aims to provide a solution to problems around massive and fast scaling needs. The Kubernetes Horizontal Pod Autoscaler cannot handle quick bursts of scale and that is the main reason why in these use cases it is not preferred. What happens then is that people pre-scale manually their applications to ensure that the anticipated load will be handled well.

Therefore, the operator can be very useful in situations with one or more of the following requirements:
* Anticipation of significantly higher than average traffic
* Multi-tenant or very diverse clusters that groups of applications need to scale together
* Same cluster applications with different scaling needs (for example, some applications should not scale, some should scale only one and some should be allowed to autoscale)
* Ability to prepare multiple applications together for an expected situation using an easy and declarative switch


## Build
The operator can be easily built by using the Makefile. By executing `make docker-build docker-push IMG=<image:tag>` you can build and push to your own registry the operator. Then you can deploy it using the make command mentioned in the next section. 
Please take a look at the Makefile to read about the rest of the commands available to build the operator in a more granular way.

## Install
The operator can be installed by executing `make deploy IMG=<image:tag>`. This will apply all manifests and the desired image and tag. For more customised deployment, you can apply directly through Kustomize the manifests in the `config/default` directory.

## Architecture
Please check the `/docs` folder for documentation related to the architecture of the PreScale Operator 

## Tests
There is a reasonably large test suite, which can be run with `make test`. This suite includes several unit tests and multiple e2e tests. We have used Ginkgo to create the e2e tests which can be executed against a real cluster or a fake one. By default, the e2e tests will expect connection to an actual cluster. Currently, we don't have an easy switch between those two and the only way to achieve this is to change the `useCluster` flag in `internal/e2e/suite_test.go`.
With every PR or merge to main branch, all tests are run in our Github workflow to ensure that everything is properly tested.


## Contributing
Please take a look at [CONTRIBUTING.md](CONTRIBUTING.md) for details on how to help out. See also the
[Architecture Guide](docs/ARCHITECTURE.md). 

## Code of Conduct
All participants in the Pre-Scaling Kubernetes Operator project are expected to comply with the code of conduct.