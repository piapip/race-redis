# Can't run this command using the make command.
# Open 8 terminals and run them manually
start-emulator: 
	# sudo redis-server /home/piapip/Desktop/redisCopy/redis2/redis.conf --daemonize yes
	# sudo redis-server /home/piapip/Desktop/redisCopy/redis3/redis.conf --daemonize yes
	cd clustering/7000 && redis-server ./redis.conf
	cd clustering/7001 && redis-server ./redis.conf
	cd clustering/7002 && redis-server ./redis.conf
	cd clustering/7003 && redis-server ./redis.conf
	cd clustering/7004 && redis-server ./redis.conf
	cd clustering/7005 && redis-server ./redis.conf
	cd clustering/7006 && redis-server ./redis.conf
	cd clustering/7007 && redis-server ./redis.conf

this:
	# For every primary that we're going to have I want to create one replica
	redis-cli --cluster create 127.0.0.1:7000 \
		127.0.0.1:7001 \
		127.0.0.1:7002 \
		127.0.0.1:7003 \
		127.0.0.1:7004 \
		127.0.0.1:7005 \
		127.0.0.1:7006 \
		127.0.0.1:7007 --cluster-replicas 1

	# To see all the master and slave.
	# redis-cli -p [any available port] -c
	# Then
	# 127.0.0.1:[given port]> CLUSTER SLOTS