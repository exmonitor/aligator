# aligator
aligator is agregation service that agregates status result in es db

Purpose of this service is to aggregate `service_status` records in elasticsearch into more campact data.

example:
from 6 records:
```
1: status true,  timestamp 2019-01-08 11:00:00
2: status true,  timestamp 2019-01-08 11:01:00
3: status false, timestamp 2019-01-08 11:02:00
4: status true,  timestamp 2019-01-08 11:03:00
5: status true,  timestamp 2019-01-08 11:04:00
6: status true,  timestamp 2019-01-08 11:05:00
```
to
```
1: status true,  from 2019-01-08 11:00:00 to 2019-01-08 11:01:00, aggregated records: 2
2: status false, from 2019-01-08 11:02:00 to 2019-01-08 11:02:00, aggregated records: 1
3: status true,  from 2019-01-08 11:03:00 to 2019-01-08 11:05:00, aggregated records: 3
```
