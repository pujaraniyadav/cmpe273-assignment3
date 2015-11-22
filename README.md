# cmpe273-assignment3

## Steps to compile

```
git clone https://github.com/pujaraniyadav/cmpe273-assignment3.git
cd cmpe273-assignment3
export GOCODE=$PWD/gocode
go build
```

## Steps to start the REST server

```
./uber-app
```
It will start the REST server in :12345

## Steps to test

1) Create Trip
```
export DATA='{"starting_from_location_id":"1", "location_ids": [ "2", "3", "4", "5" ]}'
curl -v -X POST -d "$DATA" http://localhost:12345/trips | python -m json.tool
```

2) Lookup Trip 
```
curl -v -X GET http://127.0.0.1:12345/trips/0 | python -m json.tool
```

3) Take Trip
```
curl -v -X PUT http://localhost:12345/trips/0/request | python -m json.tool
```

