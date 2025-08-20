# sync
Sync Firebird to MySQL


## Crontab

```
*/5 * * * * /bin/sh -c 'cd /usr/home/josemario/sync && echo "$(date "+[\%Y-\%m-\%d \%H:\%M:\%S]") Running sync-freebsd" >> sync.log && ./sync-freebsd >> sync.log 2>&1'
```