# Dockerutils

Go package to provide helper functions to the Docker API.

## Note

Due to providing an interface that accepts Docker types, this package does not
vendor any Docker packages and is subject to changes to the Docker API.
This is done to prevent version conflicts between this package and 
projects that depend on it.
