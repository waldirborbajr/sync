# sync
Sync Firebird to MySQL


## Crontab

```
*/5 * * * * /bin/sh -c 'echo "$(date "+[\%Y-\%m-\%d \%H:\%M:\%S]") Running sync-freebsd" >> /usr/home/josemario/sync/sync.log && /usr/home/josemario/sync/sync-freebsd >> /usr/home/josemario/sync/sync.log 2>&1'
```