# Race condition with Redis.

- The function `normalGetAndSet` only Get and Set. So if a bunch of Goroutine comes to it at the same time, it's dead because of racing problem.
- The other two `transactionGetAndSet` and `transactionGetAndSetWithVersion` use WATCH to check if the given key is updated before the transaction is executed. If at least one of the given keys is updated, abort the transaction.
And because we abort the transaction, we'll need to restart until it's successfully executed.
- So if you execute, because `normalGetAndSet` has racing problem, it's a miracle if the final value reaches to 10, whereas the other 2 `transactionGetAndSet` and `transactionGetAndSetWithVersion` will consistenly reach the value of 10 because of the WATCH operation keeping the data racing problem in check. 
- `transactionGetAndSetWithVersion` adds an addition key which is the version key, so redis can watche that version key instead of the whole chunk of CacheData. If the CacheData is too big, having the version key can be much better. Say if we have 1 million keys, and 1 version for each key, each version is just a number, say 8 bytes, so for 1 million key, we'll sacrifice 8 MB temporarily to have better performance, which is not a bad thing because Redis doesn't have much problem with an extra 8 MB. (I think xD)
