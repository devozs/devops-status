*Data Source* spesification.
now we`ll handle the data sources and we`ll make it more strict in order to reduce user inputs errors.
we also need to consider if its Operational or Qos.
once Adapter and / or Qos are selected the relevant inputs need to be changed.
whenever possible need to handle user input verification (i.e. if http and input field is url need a regex to verify it).

## FE
**Adapter**
1. HTTP - its basically a curl command. the user should give a URL and the result are either 200OK or failed.
we do need to allow the user to add its relevant curl (most popular) args
2. Prometheus - once slected need to show auth fields + test connectivity to the target prometheus.
once connected to this prometheus data source, allow the user to place its promql and again to run and test it so he can see the results.
if needed use a lib for prometheus connection.
the result here is either true / false, to indicate if Operational or not
3. Kubernetes - user will select the k8s version (As its spesific since API might be changed), we suport from 1.26 to the current latest version.
then the user need to select what k8s API is needed, currently we support the basic resources like pod, job, secrets etc. later we`ll add more types, for a name and namespace.
the call is Operational if resource is valid, avaliable and in ready state.
also here we need the user to test its data source, and as a temp binding he need to select k8s enviorment (if exists), important - k8s env can only be selected (for testing) if its version match the selected data source k8s target version.
4. CLI - this one relevant to a k8s env and needs shell context to run.
so first need to create a pod / job on the target k8s (devops-status-system ns) with unix OS.
the user should be able to run any unix commands, such as curl, apt etc. and to run the installed tools.
the expectation from the user is to evantuly set an execution that will return true or false to indicate if its operational or not.

**QoS**
1. HTTP - same as defined for adapter (defenition and testing) but here we need the user to reurn exactly three levels - red, yellow or green according to the metrics timing it took the curl call (in ms) he defined per red, yellow or green. 
2. Prometheus - same as in Adapter for auth and auth test + once connected to this prometheus data source, allow the user to place its promql and again to run and test it so he can see the results.
but here we need the user to reurn exactly three levels - red, yellow or green according to the metrics he defined per red, yellow or green.
3. Kubernetes - same as defined for adapter (defenition and testing) but here we need the user to reurn exactly three levels - red, yellow or green according to the metrics timing it took the k8s call (in ms) he defined per red, yellow or green. 
4. CLI - same as in Adapter. but the user need to give three executions, one per level red, yellow or green (the user is responsiable to do the metrics calc per level)

For both Adapter or QoS need to allow verify button that will trigger it, but its important that it can only done against actual enviorment so for the testing part the user must (temp) select an enviorment and then only to trigger the test (its like a temp binding just to verify the data source).

## Comments
- Config (JSON) and Timeout (ms) fields are redundent, we completly changed it as mentioned above.
- Make it user friendy and fully testable so user will fail on design time rather on run time

## BE
all the above is first applied for the UI and then need to be implemented at the BE to actually make the calls.
once once valid and tested (by the user on the UI) we`ll persist the datasource to the DB relevant tables.