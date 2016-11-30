[![Build Status](https://travis-ci.org/micromdm/command.svg?branch=master)](https://travis-ci.org/micromdm/command)
[![GoDoc](https://godoc.org/github.com/micromdm/command?status.svg)](http://godoc.org/github.com/micromdm/command)


The command service accepts command requests and creates MDM Payloads which can be processed by the device. 
The command service does not communicate with devices directly. Instead it serializes the payload and sends it on a message queue for other services to consume.

The Command Service can be deployed as both a library and a standalone service. 

The current implementation of the command service uses [BoltDB](https://github.com/boltdb/bolt#bolt---) to archive events and [NSQ](http://nsq.io/overview/design.html) as the message queue, both of which can be embeded in a larger standalone program. 

# Architecture Diagram
![mdm commandservice](https://cloud.githubusercontent.com/assets/1526945/20735521/9c2e3192-b66e-11e6-806f-4269c406fb80.png)

Example request/response:

```
POST /v1/commands HTTP/1.1
Content-Type: application/json; charset=utf-8
Host: localhost:8080
Connection: close
User-Agent: Paw/3.0.9 (Macintosh; OS X/10.11.6) GCDHTTPRequest
Content-Length: 188

{
    "request_type":"InstallApplication",
    "udid":"184012D9-753A-5DFC-8149-5C9AF257629F",
    "manifest_url":"https://mdm.example.com/repo/manifests/munkitools-2.5.1.2637.plist",
    "management_flags":1
}



HTTP/1.1 201 Created
Date: Wed, 30 Nov 2016 01:21:33 GMT
Content-Length: 290
Content-Type: text/plain; charset=utf-8
Connection: close

{
  "payload": {
    "CommandUUID": "a00258bc-b1d5-4c7e-addb-9c2215eb9c0f",
    "Command": {
      "request_type": "InstallApplication",
      "manifest_url": "https://mdm.example.com/repo/manifests/munkitools-2.5.1.2637.plist",
      "management_flags": 1,
      "options": {}
    }
  }
}
```

Example mdm Payload plist stored in the Event:
```
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>Command</key>
    <dict>
      <key>ManagementFlags</key>
      <integer>1</integer>
      <key>ManifestURL</key>
      <string>https://mdm.example.com/repo/manifests/munkitools-2.5.1.2637.plist</string>
      <key>Options</key>
      <dict></dict>
      <key>RequestType</key>
      <string>InstallApplication</string>
    </dict>
    <key>CommandUUID</key>
    <string>a00258bc-b1d5-4c7e-addb-9c2215eb9c0f</string>
  </dict>
</plist>
```



