# PreScaling Operator Roadmap

## What is a Roadmap and why do we need one
A roadmap is a high-level representation of the vision and direction of the product offering over time. A roadmap communicates the why and what behind what we're building.

Our product roadmap is where you can learn about what features we're working on, what stage they're in, and when we expect to bring them to you.

Currently, the main sources that feed the roadmap are the supporting architecture, goals, limitations and README docs as well as real use cases.


## Guide to the roadmap
Every item on the roadmap is an issue, with a label that indicates each of the following:

A release phase that describes the next expected phase of the roadmap item. See below for a guide to release phases.

A feature area that indicates the area of the product to which the item belongs. For a list of current product areas, see below.

Once a feature is delivered, the shipped label will be applied to the roadmap issue and the issue will be closed with a comment linking to the relevant Changelog post.

## Feature release phases
Release phases indicate the stages that the product or feature goes through, from early testing to general availability.

**alpha**: Primarily for testing and feedback
Limited availability, requires pre-release agreement. Features still under heavy development, and subject to change. Not for production use, and no documentation, SLAs or support provided.

**beta**: Publicly available in full or limited capacity
Features mostly complete and documented. Timeline and requirements for GA usually published. No SLAs or support provided.

**ga**: Generally available to all customers
Ready for production use with associated SLA and technical support obligations. Approximately 1-2 quarters from Beta.

Some of our features may still be in the exploratory stages, and have no timeframe available. These are included in the roadmap only for early feedback. These are marked as follows:

**in design**:
Feature in discovery phase. We have decided to build this feature, but are still figuring out how.

**exploring**:
Feature under consideration. We are considering building this feature, and gathering feedback on it.


## Milestones for 2021

### Q1: Jan-Mar
* Support of all CRDs and their main functionality
* Support of Kubernetes and Openshift workloads
* E2e and unit tests
* Build pipeline

### Q2: Apr-Jun
* Support of scaler type 
* Support of rate limiting
* Resource quota check
* Admission controller
* Dry-run feature

### Q3: Jul-Sep
* Add watchers for replicaset and daemonset
* Support of scheduled scaling
* Support of multiple scalers and/or scaling multiple namespaces together
* UI

### Q4: Oct-Dec
* Buffer for potential milestone shifts
* TBD
