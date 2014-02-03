# decloud

*DEC-entralized-CLOUD*

Decloud uses bitcoin to provide payments and scarce identity for a decentralized cloud.

## Background

Open source software has been a boon to developers around the world. Many pieces of closed source software have an open source equivalent: there is Linux to Windows, Mozilla to Internet Explorer, gcc to Visual Studio, and so on.

In the cloud world, things are different. We have many proprietary providers: Amazon Web Services, Dropbox, Google Cloud, GitHub, and so on, however, there is no widely used concept of "open" cloud software. Proprietary, locked in, non-interoperable systems are the standard for cloud software.

Naively, one might expect that if you released all the source code to Dropbox, that would be your "open" cloud right there. But this is not the case. The point of cloud software is that someone else provides the service, so having the source code is completely besides the point. Anything that requires owning hardware to use it is not a cloud solution for our purposes.

An open cloud system would really be a marketplace for cloud services. Buyers would be able to connect to the network to purchase services, and sellers would be able to connect to the network to provide services.

Bitcoin is an enabler for such a technology in two ways. First, it is a payment system that supports micropayments, is international, and open and decentralized. Because an open cloud system is a marketplace, there needs to be a medium of exchange, and bitcoin provides that.

Second, it provides a basis for scarce identity, which can be used to mitigate the effects of bad actors in the network. In a marketplace, there is money to be made by cheating, but by attaching a cost to making an identity, we can make cheating unprofitable. This can be as simple as, "I will only deal with nodes who can prove ownership of at least one bitcoin."

The decentralized cloud below has two components. First, there is the OpenCloud protocol, which can be thought of as an RPC protocol with payment. Second, there is the decloud client/server, which the first (and, for the immediate future, only) implementation of the OpenCloud protocol.

## Goal

In the long run, the goal is to provide an open cloud platform that is an alternative to proprietary cloud services.

In the short run, a more limited goal is useful. That goal is to provide a cheap, decentralized storage system that you can pay for with bitcoin. It should be less expensive than Dropbox, but provide comparable availability and throughput. This goal can be used as a measuring stick to evaluate how successful the decloud project is.

## OpenCloud protocol

The OpenCloud protocol defines a way for a client to make an RPC request to a server, and attach a payment, or promise of payment, to that request.

### OpenCloud Requests

Requests follow the format below. In theory, only **service**, **method**, and **args** fields are required for all requests, but the default configuration of decloud servers will also require **id**, **sig**, and **nonce**, and will require payment on many requests.

* **id**: Comma separated list of strings representing the client's identity credentials. For a given request, a node may present zero or more credentials. Currently, bitcoin addresses and OpenCloud addresses are supported as credentials.
* **sig**: Comma separate list of digital signatures. For every identity credential, one signature must be provided to prove ownership of the private key corresponding to the credential.
* **nonce**: To mitigate replay attacks, servers may request that the client include a nonce.
* **service**: The service that the client wants to access, eg. "storage" or "sha256"
* **method**: The method that the client wants to execute, eg. for a storage service, "put" or "get", or for a sha256 service "hash"
* **args**: Arguments to the method being called, eg. "storage.put(binary-blob)" or "storage.get(id)"
* **payment-type**: { none | attached | defer }
	* **none**: Request a method call for free
	* **defer**: Client promises to pay after some threshold is met, probably based on one of: time, accrued value, or service completion
	* **attached**: Payment transaction is attached. This can be used to either pay for the service being requested (client bears entire counter-party risk), or to make good on deferred payments.
* **payment**: Depends on payment-type, see table below.
* **body**: Any additional data for this request. Typical use might be binary blob for a storage "PUT" style request.

Some additional data on **payment-type** and **payment** fields:

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
* **status**: See below
* **body**: If successful call, the results. Exact format is service specific.

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

### Identity credentials

An identity credential in the OpenCloud protocol corresponds to ownership of a private key. Initially, two kinds of credentials will be understood:

1. Bitcoin addresses. For any request, a client may include proof that it controls certain bitcoin addresses. The server can examine the blockchain to determine the balance of that address, how long it has held the balance, what miner fees are associated with that address, etc. How this information is used is based on the servers <a href="#policy">policy</a>.
2. OpenCloud ID's. This is a simple private/public key pair. The intention, though, is that they will be more stable than using bitcoin addresses, since people may want to spend their bitcoins. By default, the decloud server will associate reputation information with OpenCloud ID's, not with bitcoin addresses, and will interact with OpenCloud ID's that it trusts regardless bitcoin identity credentials.

In brief, bitcoin balances are meant to be an initial guard against spam/bad actors, and OpenCloud ID's are meant to be your identity in the system.

### Reputation

Reputation encompasses:

* What you think of others
* What others think of you
* What others think of others

For **what you think of others**: when you interact with any node in the OpenCloud system, that node will make promises and then fulfill (or not) those promises. Your opinion of the other node is how well they did (or did not) fulfill their promises.

In the case of a server, you will store deferred payment promises made by clients, and match that against the actual payment received. The "defer-threshold" field is helpful in evaluating an honest client who's payment is not yet due, vs. a cheating client.

In the case of a client, you will store promises of a level of service (eg. storing data for 1 day, making it available at 10MB/s) and compare it to the actual service received (eg. data was stored for 1 day, actual throughput was 8MB/s).

For **what others think of you**: much like you are keeping records on other nodes, they are also keeping records on you.

For **what others think of others**: it would be useful to take advantage of the repuation knowledge of the entire network, but lying makes this difficult. At present, the idea is the treat reputation like a service, and evaluation it in the same way that you would evaluate any other service. For example, if a node consistently gives opinions in line with your own experience, you trust that nodes opinions in general. An implementation of the [EigenTrust algorithm](http://en.wikipedia.org/wiki/EigenTrust) may be used.

For reputation knowledge to be effective, standard units for level of service and pricing must be defined. See the section below on [service templates](#template) for more information.

### Serialization format

To be determined

### OpenCloud protocol over HTTP

To be determined, but will probably be supported at some point


## Decloud - Server

A decloud sever serves requests received the OpenCloud protocol, in the same away that an Apache server serves HTTP requests.

### Request processing

Decloud servers handle an incoming request in the following fashion:

* Is the request valid?
	* Are sigs valid?
	* Is the nonce valid?
	* Is the service available?
	* Is the method available?
* Access controls, reputation, payment
	* Based on credentials, do we grant access?
	* Based on reputation, do we serve this request?
	* Based on payment, do we serve this request?
* If payment is attached, update reputation information
* Pass off to service

Decloud clients send requests, and handle responses, in the following fashion:

* Send request
	* Set service, method, args, nonce, body, and payment
	* Sign request
	* Send request
* Response handling
	* If **ok**: exit
	* If **client-error**: report error and exit
	* If **server-error**: report error and exit
	* If **request-declined**:
		* If **refresh-nonce**: re-send request with new nonce
		* If **payment-declined**:
			* If **too-low**: based on bidding strategy, either increase payment or exit
			* If **no-defer**: based on bidding strategy, either switch to **attached" payment or exit

### Services

A decloud server can choose to run any number of services.

<a name="template"></a>
#### Template

Defining a new service follows a template.

Decloud services provide the following standard method calls:

* **service.info**: version information
* **service.methods**: will provide a list of available methods
* **service.quote(method_name, units_of_service)**: cost of a method call in some unit. The unit of cost is determined by each service, and may have multiple dimensions (eg. space and duration for storage, or CPU hours and RAM for computation)

Additionally, every service defines pricing and service units. In reputation logging, the client will record the promised and actual level of service in standard units. The standard pricing units are used by the **market** service to distribute pricing information across the network.

<a name="policy"></a>
#### Policies 

Decloud services can be configured to follow policies. These can be thought of a pricing strategies, or client selection strategies. To use a real world example: a high class bar may decide to be exclusive, but also also rich clients to run up tabs of $1000+, while a working class bar may decide to allow anyone in, but only allow tabs up to $100.

Policies exist in the following scopes:

* **global** policies: apply to everything
* **service** policies: apply only to specific services
* **method** policies: apply only to specific methods
* **credential** policies: apply only to specific identity credentials
* To be determined: how to define reputation based scopes

The following policy commands are standard across all services:

* **allow**: allow access
* **deny**: deny access
* **min-payment**: require at least this much payment
* **rate-limit**: limit the number of queries per second allowed
* To be determined: policy commands for handling defered payments
* To be determined: additional policy commands

Service and method specific configuration may also be supported.

#### To be determined

* Namespacing

### Core services

Possibly alternative names: "meta services" or "system services." These are services that are not inherently useful, but support the decloud system, and are used by actually useful services.

* **info**: basic information about a node
* **peers**: share information about peers
* **repuation**: share my reputation data
* **market**: share my knowledge of market prices

Additional details to be determined.

### Application services

The initial application service will be storage.

Following application services will be a distributed file system, and computation.

Additional details to be determined.

## Decloud - Client

A decloud client has two functions:

1. Make requests to servers over the OpenCloud protocol. This will initially be a command-line tool, comparable to **wget**.
2. Fulfill deferred payments on services that span longer time frames. This does not have a straight analogy in HTTP. It will initially be a daemon process.

Additioanl details to be determined

## Additional notes

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
* Bid acceptance strategy

Components of a decloud **client**:

* Defered payment fulfillment
* Long-term service auditing (eg. storage)
* Bidding strategy
