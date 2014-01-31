decloud
=======

*DEC-entralized-CLOUD*

Decloud uses bitcoin to provide payments and scarce identity for a decentralized cloud.

Background
-------

Open source software has been a boon to developers around the world. Much closed source software has an open source equivalent: there is Linux to Windows, Mozilla to Internet Explorer, gcc to Visual Studio, and so on.

In the cloud world, things are different. We have many proprietary providers: Amazon Web Services, Dropbox, Google Cloud, GitHub, and so on. However, there is no widely used concept of "open" cloud software. Proprietary, locked in, non-interoperable systems are the standard for cloud software.

Naively, one might expect that if you released all the source code to Dropbox, that would be your "open" cloud right there. But this is not the case. The point of cloud software is that someone else provides the service, so having the source code is completely besides the point. Anything that requires owning hardware to use it is not a cloud solution for our purposes.

An open cloud system would really be a marketplace for cloud services. Buyers would be able to connect to the network to purchase services, and sellers would be able to connect to the network to provide services.

Bitcoin is an enabler for such a technology in two ways. First, it is a payment system that has all of the following properties:

1. Supports micropayments
2. International
3. Open

Second, it provides a basis for scarce identity, which can be used to mitigate the effects of bad actors in the network.

Below is a working draft design document of the OpenCloud protocol, and the decloud client/server.

OpenCloud protocol
--------

The decloud client and server communicate through the OpenCloud protocol, in the same way that a web browser and Apache communicate over HTTP.

Below is a draft of the protocol.

### OpenCloud Requests

* **id**: Comma separated list of strings representing the client's identity credentials. For a given request, a node may present zero or more credentials. Currently, bitcoin addresses and OpenCloud addresses are supported as credentials.
* **sig**: Comma separate list of digital signatures. For every identity credential, one signature must be provided to prove ownership of the private key corresponding to the credential.
* **nonce**
* **service**
* **method**
* **args**
* **payment-type**: { none | attached | defer }
* **payment**: Depends on payment-type
* **body**

The **payment-type** and **payment** fields:

<table>
	<tr>
	<th style="white-space:nowrap;">payment-type</th>
	<th>payment</th>
	</tr>

	<tr>
	<td>none</td>
	<td>empty</td>
	</tr>

	<tr>
	<td>attached</td>
	<td>
	[currency] [amount] [txn]

	<ul>
	<li><b>currency</b>: string, typicaly BTC, USD, EUR, etc.</li>
	<li><b>amount</b>: floating point number, the amount of the payment</li>
	<li><b>txn</b>: base64 encoded transaction in the payment amount</li>
	</ul>
	</td>

	<tr>
	<td>defer</td>
	<td>
	[currency] [amount] [id] [optional: defer-threshold]
	<ul>
	<li><b>currency</b>: string, typicaly BTC, USD, EUR, etc.</li>
	<li><b>amount</b>: floating point number, the amount of the payment</li>
	<li><b>id</b>: id with which to associate this defered payment. Server must not accept ID unless it has provided a valid signature on this request.</li>
	<li><b>defer-threshold</b>: Optional, if included, describes to the server the trigger for fulfilling the defered payment</li>
	</ul>
	</td>

	</tr>
</table>

Additional **payment-type**'s may be supported in the future, such as micropayment channels.

### OpenCloud Responses

* **id**: Same as request
* **sig**: Same as request
* **nonce**: Same as request
* **status**
* **acceptable-payment**
* **body**

The **status** field:

* **ok**: Equivalent of 2xx for HTTP
* **client-error**: Equivalent of 4xx for HTTP
  * **bad-request**
  * **invalid-signature**
  * **service-unsupported**
  * **method-unsupported**
* **server-error**: Equivalent of 5xx for HTTP
* **request-declined**: A valid request that was declined.
  * **refresh-nonce**: 
  * **payment-declined**: Optional detail below
    * **too-low**: Payment is too low
    * **no-defer**: Defer payment is not accepted
    * **acceptable-payment**: optional, server may indicate an acceptable payment for this request [payment type] [currency] [amount]

### Serialization format

To be determined

### OpenCloud protocol over HTTP

To be determined, but will probably be supported

Decloud
-------

A decloud sever serves requests received the OpenCloud protocol, in the same away that an Apache server communicates via HTTP. Decloud also encompasses a client implementation, which is comparable to **wget**.

Below is a rough outline of the components of the decloud client and server.

Components **shared** between decloud clients and servers:

* Identity credentials
  * OpenCloud ID
  * Bitcoin wallet (addresses) as credentials
* Reputation management
  * Record collection
  * Aggregation
  * Peer provided reputation handling
* OpenCloud protocol support

Components of a decloud **server**:

* Access control
  * Service/method granularity
  * May reference identity credentials
* Specific service implementations
* Deferred payment tracking

Components of a decloud **client**:

* Defered payment fulfillment
* Long-term service auditing (eg. storage)

### Server request processing

Decloud servers handle an incoming request in the following fashion:

* Is the request valid?
	* Are sigs valid?
	* Is the nonce valid?
	* Is the service available?
	* Is the method available?
* Access controls, reputation, payment
	* Based on credentials, do we grant access?
	* Based on reputation, do we serve this request?
	* Based on payment, do we serve this requset?
* Pass of to service

Decloud clients send requests, and handle responses, in the following fashion:

* Request
	* Set service, method, args, nonce, body, and payment
	* Sign request
	* Send request
* Response handling
	* If **ok**: exit
	* If **client-error**: report error and exit
	* If **server-error**: report error and exit
	* If **request-declined**:
		* If **refresh-nonce**: re-send request with new nonce
		* If **payment-declined"
			* If **too-low**: based on bidding strategy, either increase payment or exit
			* If **no-defer**: based on bidding strategy, either switch to **attached" payment or exit
