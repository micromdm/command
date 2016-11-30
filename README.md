The command service accepts command requests and creates MDM Payloads which can be processed by the device. 
The command service does not communicate with devices directly. Instead it serializes the payload and sends it on a message queue for other services to consume.

The current implementation of the command service uses BoltDB to archive events and NSQ as the message queue, both of which can be embeded in a larger standalone program. 
