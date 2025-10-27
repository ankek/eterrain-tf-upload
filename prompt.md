i want to create backend service for terraform provider.
Use Go Lang 1.25 for development.
used port is 7777
we will use existing TF provider. Do not develop TF code and TF provider here.

backend service:
for authentification we will use next credential details: orgid, apikey. Where "orgid" is Org ID - a unique Organization ID (UUID) of organization that uploading data with TF provider . "apikey" is a unique token of organization that auth in service

backend service should support next TF provider configuration 
  url = "http://127.0.0.1:7777"         
  org_id = "11111111-2222-3333-4444-555555555555"
  apikey = "demo-api-key-12345"

Terraform generate data within TF provider and upload it to this backend service. 
backend service get such data from terraform provider and store it in {Org ID}.csv file in historical order.
{Org ID}.csv may have many uploads. all of them should be saved in one {Org ID}.csv file




here is logical structure of uploaded data. modify API of this service to follow this structure and return me example of  API call. i want to test data upload
{DIGITALOCEAN
    {compute
        {digitalocean_droplet
            {name    = "web-1" 
                { size    = "s-1vcpu-1gb"}
            }
            {name    = "web-2" 
                { size    = "s-2vcpu-2gb"}
            }
        }
        {digitalocean_droplet_autoscale
            {name = "terraform-example"
                {min_instances             = 10,  max_instances             = 50, size               = "c-2"}
            }
        }
    }
}

 API to support the hierarchical structure. The structure i want is:

  Provider (e.g., DIGITALOCEAN)
  └── Category (e.g., compute)
      └── Resource Type (e.g., digitalocean_droplet)
          └── Resource Instance (e.g., name: "web-1")
              └── Properties (e.g., size: "s-1vcpu-1gb")

CURL example 
 curl -X POST "http://127.0.0.1:7777/api/v1/upload" \
    -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
    -H "X-API-Key: demo-api-key-12345" \
    -H "Content-Type: application/json" \
    -d '{
      "provider": "DIGITALOCEAN",
      "category": "compute",
      "resource_type": "digitalocean_droplet_autoscale",
      "resource_name": "terraform-example",
      "properties": {
        "min_instances": 10,
        "max_instances": 50,
        "size": "c-2"
      }
    }'




modify service to store Org-ID and API-Key in ./auth.cfg file. Use values for authentification from there. structure of file is 

[11111111-2222-3333-4444-555555555555]
demo-api-key-12345
demo-api-key-12347
demo-api-key-12349




security-compliance -> backend-api-security


"lets extend functionality. i want to store data in csv files and in mysql db. Data should be stored in DB with name "data", one table per organization with {Org ID} like "11111111-7777-3333-4444-555555555555" and exact same structure as for csv file with timestamp,org_id,data as columnt in such table. i want to run it in docker-compose like 
```
version: '3.8'
services:
  eterrain-tf-backend:
    image: 8589344fc185
    container_name: terraform-backend
#    ports:
#      - "${HOST_PORT:-7777}:7777"
    environment:
      - VIRTUAL_HOST=tf.eterrain.ee
      - LETSENCRYPT_HOST=tf.eterrain.ee
      - LETSENCRYPT_EMAIL=ankek777@gmail.com
      - HOST=0.0.0.0
      - PORT=7777
      - STORAGE_TYPE=${STORAGE_TYPE:-csv}
      - STORAGE_PATH=/app/data
      - ENABLE_TLS=${ENABLE_TLS:-false}
      - TLS_CERT_FILE=${TLS_CERT_FILE:-}
      - TLS_KEY_FILE=${TLS_KEY_FILE:-}
    volumes:
      # Persist data directory
      - ./data:/app/data
      # Mount auth configuration (read-only)
      - ./auth.cfg:/app/auth.cfg:ro
      # Optional: Mount TLS certificates if needed
      # - ./certs:/app/certs:ro
    restart: unless-stopped

  db:
    image: mysql:8.4
    restart: always
    environment:
      MYSQL_DATABASE: exampledb
      MYSQL_USER: exampleuser
      MYSQL_PASSWORD: yrA7MpQwKVxAfWP6XhZ54mtG7qZZfVzHe7mULWmFsgzY6krSUZ
      MYSQL_RANDOM_ROOT_PASSWORD: '1'
    volumes:
      - ./db:/var/lib/mysql

volumes:
  wordpress:
  db:

networks:
  default:
    external:
      name: nginx-proxy
```
" backend-architect → database-architect → frontend-developer → test-automator → security-auditor