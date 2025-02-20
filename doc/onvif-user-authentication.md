# Onvif User Authentication

According to the Onvif user authentication flow, the device service shall:
* Implement WS-Usernametoken according to WS-security as covered by the core specification.
* Implement HTTP Digest as covered by the core specification.

The spec can refer to https://www.onvif.org/specs/core/ONVIF-Core-Specification.pdf

![onvif-user-authentication](images/onvif-user-authentication.jpg)

## Usage
The user need to define the **AuthMode** and **SecretName**, and device service will send SOAP action with **WS-Usernametoken** or **Digest header**.

For example:
```yaml
[[DeviceList]]
Name = "test-camera"
ProfileName = "camera"
Description = "HIKVISION camera"
  [DeviceList.Protocols]
    [DeviceList.Protocols.Onvif]
    Address = "192.168.12.123"
    Port = 80
    AuthMode = "usernametoken"
    SecretName = "credentials001"
```

The AuthMode can be:
* digest
* usernametoken
* both
* none

SecretName should contain:
* username
* password

For development purpose, we can define the secrets in the configuration.toml
```
[Writable]
...
  [Writable.InsecureSecrets]
    [Writable.InsecureSecrets.Camera001]
    secretName = "credentials001"
      [Writable.InsecureSecrets.Camera001.SecretData]
      username = "administrator"
      password = "Password1"
    # If having more than one camera, uncomment the following config settings
    [Writable.InsecureSecrets.Camera002]
    secretName = "credentials002"
      [Writable.InsecureSecrets.Camera002.SecretData]
      username = "administrator"
      password = "Password1"
```

## WS-Usernametoken
When the Onvif camera requires authentication through WS-UsernameToken, the device service must set user information with the appropriate privileges in WS-UsernameToken. 

This use case contains an example of setting that user information using GetHostname.

WS-UsernameToken requires the following parameters:
* Username – The user name for a certified user.
* Password – The password for a certified user. According to the ONVIF specification, Password should not be set in plain text. Setting a password generates PasswordDigest, a digest that is calculated according to an algorithm defined in the specification for WS-UsernameToken:
  Digest = B64ENCODE( SHA1( B64DECODE( Nonce ) + Date + Password ) )
* Nonce – A random string generated by a client. 
* Created – The UTC Time when the request is made.

For example:
```shell
curl --request POST 'http://192.168.56.101:10000/onvif/device_service' \
    --header 'Content-Type: application/soap+xml' \
    -d  '<?xml version="1.0" encoding="UTF-8"?>
    <soap-env:Envelope xmlns:soap-env="http://www.w3.org/2003/05/soap-envelope" ...>
        <soap-env:Header>
            <Security xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">
                <UsernameToken>
                    <Username>administrator</Username>
                    <Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest">
                      +HKcvc+LCGClVwuros1sJuXepQY=
                    </Password>
                    <Nonce EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary">
                      w490bn6rlib33d5rb8t6ulnqlmz9h43m
                    </Nonce>
                    <Created xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">
                      2021-10-21T03:43:21.02075Z
                    </Created>
                </UsernameToken>
            </Security>
        </soap-env:Header>
        <soap-env:Body>
            <tds:GetHostname>
            </tds:GetHostname>
        </soap-env:Body>
      </soap-env:Envelope>'
```

The spec can refer to https://www.onvif.org/wp-content/uploads/2016/12/ONVIF_WG-APG-Application_Programmers_Guide-1.pdf

You can inspect the request by network tool like the Wireshark:
![onvif-user-authentication-usernametoken](images/onvif-user-authentication-usernametoken.jpg)

## HTTP Digest
The Digest scheme is based on a simple challenge-response paradigm and the spec can refer to https://datatracker.ietf.org/doc/html/rfc2617#page-6

The authentication follow can be illustrated as below:
1. The device service sends the request without the acceptable Authorization header.
2. The Onvif camera return the response with a "401 Unauthorized" status code, and a WWW-Authenticate header.
   - The WWW-Authenticate header contains the required data
      - qop: Indicates what "quality of protection" the client has applied to the message.
      - nonce: A server-specified data string which should be uniquely generated each time a 401 response is made. The onvif camera can limit the time of the nonce's validity.
      - realm: name of the host performing the authentication
   - And the device service will put the qop, nonce, realm in the header at next request
3. The device service sends the request again, and the Authorization header must contain:
   - qop: retrieve from the previous response
   - nonce: retrieve from the previous response
   - realm: retrieve from the previous response
   - username: The user's name in the specified realm.
   - uri: Request uri
   - nc: The nc-value is the hexadecimal count of the number of requests (including the current request) that the client has sent with the nonce value in this request.
   - cnonce: A random string generated by a client.
   - response: A string of 32 hex digits computed as defined below, which proves that the user knows a password.
     - MD5( hash1:nonce:nc:cnonce:qop:hash2)
       - hash1: MD5(username:realm:password)
       - hash2: MD5(POST:uri)
   
4. The Onvif camera return the response with a "200 OK" status code


Inspect the request by the Wireshark:
![onvif-user-authentication-flow](images/onvif-user-authentication-flow.jpg)
