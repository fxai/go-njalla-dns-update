## DNS updater for Njalla

This is a small Njalla DNS updater. 
It will grab the current external IP address and will update the defined DNS entries with that IP address.

## Environment Variables

You need to define 2 environment variables inside of your container:

### $njalla_update

Defines a json formatted list of subdomains that you want to automatically update.

json structure:
```
{
  "update": [
    {
      "sub": "subdomain",
      "domain": "domain"
    }
  ]
}
```

Example with two DNS records to update
```
export njalla_update = '{"update": [{"sub": "subdomain1", "domain": "domain"}, {"sub": "subdomain2", "domain": "domain"}]}'
```

### $njalla_update_interval 
Defines the update interval in seconds.

Example that will trigger the update all 10mins
```
export njalla_update_interval = "600" 
```

## API Token
The API token is expected to be a file inside the container. It should only contain the API key. Expected path: /vault/secret/api.txt

Example
```
$cat /vault/secrets/api.txt
1234567890topsecure
```
