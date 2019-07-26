# Demo

Below the steps to run the demo.
We assume you have a 4.x cluster in AWS.

deploy hive:

```shell
mkdir -p $GOPATH/src/github.com/openshift
cd $GOPATH/src/github.com/openshift
git clone https://github.com/openshift/hive
cd hive
make deploy
```

deploy the federation-lifecycle-operator, see [here](./README.md#Installation)

deploy the federation configuration:

```shell
export BASE_DOMAIN=sandbox205.opentlc.com
oc new-project demo
oc process -f ./test/clusterdeploymentset.yaml \
   CLUSTER_NAME=demo \
   SSH_KEY="$(cat ~/.ssh/sshkey-gcp.pub)" \
   PULL_SECRET="$(cat ./test/pull-secret.json)" \
   AWS_ACCESS_KEY_ID="$(cat ~/.aws/credentials | grep aws_access_key_id | awk '{ print$3 }')" \
   AWS_SECRET_ACCESS_KEY="$(cat ~/.aws/credentials | grep aws_secret_access_key | awk '{ print$3 }')" \
   BASE_DOMAIN=${BASE_DOMAIN} \
   NAMESPACE=demo \
   -n demo | oc apply -f - -n demo
```

check that a `ClusterDeploymentSet` has been created and that corresponding clusters are being created:

```shell
oc get ClusterDeploymentSet -n demo
oc get ClusterDeployment -n demo
```

check that a `multiplenamespacefederation` has been created:

```shell
oc get multiplenamespacefederation -n demo
```

create a couple of namespaces that match the `multiplenamespacefederation` label selector

```shell
for ns in fns1 fns2 ; do
  oc new-project $ns
  oc label namespace $ns federation=demo
done
```

verify that `namespacefederation` instances have bee created in those namespaces:

```shell
oc get namespacefederation --all-namespaces
```

verify that `federatedconfigtypes` have been created in the federated namespaces:

```shell
oc get federatedtypeconfig --all-namespaces
```

verify that `domains` have beed created in the federated namespaces:

```shell
oc get domain --all-namespaces
```

Now wait for the clusters to be created, this can take between 30 and 45 minutes.

After the clusters have been provisioned, verify that the *cluster registry* is populated:

```shell
oc get cluster -n demo
```

Then verify that clusters have been federated in the federated namespaces:

```shell
oc get federatedclusters --all-namespaces
```

At this point we can deploy an application in one of the federated namespaces

```shell
cat ./test/federatedapp.yaml | envsubst | oc apply -f - -n fns1
```

Using the webconsole verify that the app has been deployed in all of the federated clusters

Verify that the `dnsendpoint` for the app has been created

```shell
oc get dnsendpoint -n fns1 -o yaml
```

There should be three endpoints for the name `myhttpd`.

Now verify that the name can be resolved:

```shell
nslookup myhttpd.demo-fed-${BASE_DOMAIN}
```

If you get an answer now you should be able to `curl` the application

```shell
curl http://myhttpd.demo-fed-${BASE_DOMAIN}
```

To clean up:

```shell
oc delete -f ./test/federatedapp.yaml -n fns1
oc label namespace fns1 federation-
oc label namespace fns2 federation-
oc delete namespacefederation demo -n fns1
oc delete namespacefederation demo -n fns2
sleep 20
for ns in fns1 fns2 ; do
  oc delete namespace $ns
done
oc process -f ./test/clusterdeploymentset.yaml \
   CLUSTER_NAME=demo \
   SSH_KEY="$(cat ~/.ssh/sshkey-gcp.pub)" \
   PULL_SECRET="$(cat ./test/pull-secret.json)" \
   AWS_ACCESS_KEY_ID="$(cat ~/.aws/credentials | grep aws_access_key_id | awk '{ print$3 }')" \
   AWS_SECRET_ACCESS_KEY="$(cat ~/.aws/credentials | grep aws_secret_access_key | awk '{ print$3 }')" \
   BASE_DOMAIN=${BASE_DOMAIN} \
   NAMESPACE=demo \
   -n demo | oc delete -f - -n demo
```

If any namespace gets stuck, use this script to find the resource whose finalizers are not working

```shell
for resource in $(oc api-resources -o name --no-headers=true --namespaced=true); do echo $resource; oc get $resource -n <namespace>; done
```

then use this script to remove the finalizers

```shell
oc patch <resource-type> <name> -n <namespace> -p '{"metadata":{"finalizers":[]}}'
```
