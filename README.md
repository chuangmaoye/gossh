# Gossh, implementing simple server SSH login management and SFTP file upload and download


Gossh supports Mac, Linux, and Windows (Todo);

## Function Description


*   Manager server
*   No password required to log in to the server
*   SFTP file copying, supporting local file or folder copying to the server, server file or folder copying to the local, supporting file or folder multi server copying; Support multi-threaded file downloading and file breakpoint continuation.
*   Ignoring unnecessary download files (todo)

#### Manager server

***

*   Create a configuration file `config.yml` in the installation directory, which is a global configuration file.
*   Create a configuration file `config.yml` in the relative directory. If Gossh is executed in the relative directory, the configuration file in the current directory will be prioritized.
*   Parameter with configuration file ` gossh -c file.yml`
*   Description

        addresss:
          - name: 'Server login name'
            ip: 'ip'
            password: ''
            port: 22 
            key: 'Abbreviation for Login Server'
            pem: 'Local key'
            servername: 'server name'
          - name: 'Server login name'
            ip: 'ip'
            password: ''
            port: 22 
            key: 'Abbreviation for Login Server'
            pem: 'Local key'
            servername: 'server name'

#### Password free login to server

***

```bash
gossh link [key|index]
例：
gossh link s178
gossh link 0
```

#### Copy

***

```bash
gossh cp LocalFile s178:RemotePath s179:RemotePath....
gossh cp -r LocalFile s178:RemotePath s179:RemotePath... // -r Force Overwrite
gossh cp -t 10 LocalFile s178:RemotePath s179:RemotePath.... // -t Ctrip quantity, default to 10 Ctrip
```

#### directories copying

***

```bash
gossh cp -d LocalFile s178:RemotePath s179:RemotePath....
gossh cp -r -d LocalFile s178:RemotePath s179:RemotePath... // -r Force Overwrite
gossh cp -t 10 LocalFile s178:RemotePath s179:RemotePath.... // -t Ctrip quantity, default to 10 Ctrip
```

