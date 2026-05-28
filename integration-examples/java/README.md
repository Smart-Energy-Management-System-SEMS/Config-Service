# Java usage

Compilar y ejecutar (Java 17+):

```bash
javac pe/edu/upc/configclient/ConfigClient.java
java pe.edu.upc.configclient.ConfigClient
```

Si usas Spring Boot, mueve esta logica a un `@Component` de bootstrap y parsea JSON con Jackson.
