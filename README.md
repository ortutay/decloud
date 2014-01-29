decloud
=======

*DEC-entralized-CLOUD*

Decloud uses bitcoin to provide payments and scarce identity for a decentralized cloud.

OpenCloud protocol
--------

Decloud communicates through the OpenCloud protocol.

### OpenCloud Requests

* **id**: Comma separated list of strings representing the client's identity credentials. For a given request, a node may present zero or more credentials. Currently, bitcoin addresses and OpenCloud addresses are supported as credentials.
* **sig**: Comma separate list of digital signatures. For every identity credential, one signature must be provided to prove ownership of the private key corresponding to the credential.
* **nonce**: Optional nonce
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
	<li>**currency**: string, typicaly BTC, USD, EUR, etc.</li>
	<li>**amount**: floating point number, the amount of the payment</li>
	<li>**txn**: base64 encoded transaction in the payment amount</li>
	</ul>
	</td>

	<tr>
	<td>defer</td>
	<td>
	[currency] [amount] [id]
	<ul>
	<li>**currency**: string, typicaly BTC, USD, EUR, etc.</li>
	<li>**amount**: floating point number, the amount of the payment</li>
	<li>**id**: id with which to associate this defered payment. Server must not accept ID unless it has provided a valid signature on this request.</li>
	</ul>
	</td>

	</tr>
</table>

Additional **payment-type**s may be supported in the future, such as micropayment channels.

### OpenCloud Responses

* **id**: Same as request
* **sig**: Same as request
* **nonce**: Same as request
* **status**
* **acceptable-payment**
* **body**

The **status** field:

*ok*
*error*
*payment-declined*

For *payment-declined*, the server may choose to include additional, optional information:
> **too-low**: Payment is too low
> **no-defer**: Defer payment is not accepted

The server may also choose to include a list of acceptable payment options, in the form:
> **acceptable-payment**: [payment type, {attached|defer}] [currency] [amount] [optional: defer-account] [optional: defer-threshhold]

