# Demo

Below the steps to run the demo.
We assume you have a 4.x cluster in AWS.

deploy hive:

```shell
mkdir -g $GOPATH/src/github.com/openshift
cd $GOPATH/src/github.com/openshift
git clone https://github.com/openshift/hive
cd hive
make deploy
```

deploy the federation cycle operator, see [here](./README.md#Installetion)