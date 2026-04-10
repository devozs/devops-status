# Resources, Enteties and Screen Changes
while starting to test i figured out that some of the resources and thier structure doesnt make sense.
We need to consider the following changes that will impact FE, BE and DB.

## Structure
we`ll have the following resources
### Service Provider (new)
- host
- port

## k8s-clusters (existing)
dont touch, it is designed and working perfectly

### Data Sources
we need to combine Operational and Qos, they have the same URL. The QoS is actually relevant if DS is operetional.
so need to update the DB entity to hold both as well as the "Create Data Source" modal.
When user click on Verify within the modal it will first test the URL (as already done) and also test the QoS latency (As user defined).
give color background to Green max, Yellow max and Red max and when user click on verify it will blick at the relevant level per API hit results (and per adapter type).

## Enviorment
Need to combine it with binding (and remove binding)
- enviorment will point to cluster (k8s type)
- enviorment will point to 1 or more data sources of k8s releated type
each data source will have the previous biding defenition: Interval (sec), Window, Failures to Down and Success to Recover


## Service
Need to combine it with binding (and remove binding)
- service will point to service provider (service type)
- service will point to 1 or more data sources of service releated type
each data source will have the previous biding defenition: Interval (sec), Window, Failures to Down and Success to Recover


## Backward Compatability
not needed, feel free to update code, DB etc. we are at development stage.
dont create scripts to support old data.