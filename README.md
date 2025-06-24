# Tezos Delegation Service

In this exercise, you will build a Golang service that gathers all delegations made on the Tezos protocol and exposes them through a public API. 

## Run the code 

This command will build, create the database, backfill the records and launch a public API at port 3000.
```
make run 
```
API is accessible at ```http://localhost:3000/xtz/delegations```

## Run the tests 
```
make test 
```
## Additional commands
```
make build 
```