<!-- PROJECT SHIELDS -->
[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![MIT License][license-shield]][license-url]

<!-- TABLE OF CONTENTS -->
<details>
  <summary>Table of Contents</summary>
  <ol>
    <li>
      <a href="#about-the-project">About The Project</a>
      <ul>
        <li><a href="#built-with">Built With</a></li>
      </ul>
    </li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
      </ul>
    </li>
    <li><a href="#usage">Usage</a></li>
    <li><a href="#roadmap">Roadmap</a></li>
    <li><a href="#contributing">Contributing</a>
      <ul>
        <li><a href="#testing">Testing</a></li>
      </ul>
    </li>
    <li><a href="#license">License</a></li>
    <li><a href="#contact">Contact</a></li>
    <li><a href="#acknowledgments">Acknowledgments</a></li>
  </ol>
</details>



<!-- ABOUT THE PROJECT -->
## About The Project

I needed a simple executable to explore and export data from an Oracle DB on a locked down corporate laptop/network. This was my starting point for an automated data retrieval program which I can't share publicly. This repo is public in hopes that someone else will find it useful.

Here's why:
* Oracle CLI produces broken CSV files
* Oracle Developer didn't work for me either.
* This is lightweight and simple;

Of course, no one tool will serve all projects since your needs may be different. So you may also suggest changes by forking this repo and creating a pull request or opening an issue.

<p align="right">(<a href="#top">back to top</a>)</p>



### Built With

* [Go](https://golang.org/)
* [go-ora](https://github.com/sijms/go-ora)
* [tablewriter](https://github.com/olekukonko/tablewriter)
* [progressbar](https://github.com/schollz/progressbar)

<p align="right">(<a href="#top">back to top</a>)</p>



<!-- GETTING STARTED -->
## Getting Started

To get a local copy up and running follow these simple example steps.

### Prerequisites

You will need to install Go.
* go
 <a href="https://go.dev/doc/install">Install Go</a>

### Installation

To get this up and running:

1. Clone the repo
   ```sh
   git clone https://github.com/sixcolors/orasuck.git
   ```
2. Install Go mods
   ```sh
   go mod tidy
   ```
3. Build orasuck
   ```sh
   go build -o orasuck main.go
   ```
   
   * for windows
   ```sh
   GOOS=windows GOARCH=386 go build -o orasuck.exe main.go
   ```

<p align="right">(<a href="#top">back to top</a>)</p>



<!-- USAGE EXAMPLES -->
## Usage

* Query and display (pretty table) to stdout
```sh
orasuck -server "oracle://user:pass@server/service_name" "select * from my_table"
```

* Query and export to csv file
```sh
orasuck -server "oracle://user:pass@server/service_name" -file "out.csv" "select * from my_table"
```

* Query and export to json file (auto-detected by extension)
```sh
orasuck -server "oracle://user:pass@server/service_name" -file "out.json" "select * from my_table"
```

* Query and output json to stdout
```sh
orasuck -server "oracle://user:pass@server/service_name" -json "select * from my_table"
```

* Query and output csv to stdout
```sh
orasuck -server "oracle://user:pass@server/service_name" -csv "select * from my_table"
```

<p align="right">(<a href="#top">back to top</a>)</p>



<!-- ROADMAP -->
## Roadmap

- [x] Initial Release

See the [open issues](https://github.com/sixcolors/orasuck/issues) for a full list of proposed features (and known issues).

<p align="right">(<a href="#top">back to top</a>)</p>



<!-- CONTRIBUTING -->
## Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also simply open an issue with the tag "enhancement".
Don't forget to give the project a star! Thanks again!

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

<p align="right">(<a href="#top">back to top</a>)</p>

### Testing

You can run the unit tests locally without any external dependencies:

```sh
go test -v
```

For integration testing, we can test Orasuck using Oracle Database Free Edition. For more information, see [Oracle Database Free Release Quickstart](https://www.oracle.com/database/free/get-started/#quick-start).

Run the Oracle DB container.

```sh
docker run --name oracle -p 1521:1521 -e ORACLE_PWD=Test123 container-registry.oracle.com/database/free:latest
```

Connect to the Oracle DB container.

```sh
docker exec -it oracle bash
```

Connect to the Oracle DB.

```sh
sqlplus sys@localhost:1521/FREE as sysdba
```

Create a user and grant privileges.

```sql
alter session set "_ORACLE_SCRIPT"=true;
CREATE USER test IDENTIFIED BY Test123;
GRANT CONNECT, RESOURCE TO test;
ALTER USER test QUOTA UNLIMITED ON USERS;
```

Create a table and insert some data.

```sql
CREATE TABLE test.test_table (id NUMBER, name VARCHAR2(50));
INSERT INTO test.test_table VALUES (1, 'John Doe');
INSERT INTO test.test_table VALUES (2, 'Jane Doe');
```

Grant select privileges.

```sql
GRANT SELECT ON test.test_table TO test;
```

Test the connection.

```sh
go run main.go -server "oracle://test:Test123@localhost:1521/FREE" "select * from test_table"
```


<p align="right">(<a href="#top">back to top</a>)</p>

<!-- LICENSE -->
## License

Distributed under the MIT License. See `LICENSE.txt` for more information.

<p align="right">(<a href="#top">back to top</a>)</p>



<!-- CONTACT -->
## Contact

@sixcolors

Project Link: [https://github.com/sixcolors/orasuck](https://github.com/sixcolors/orasuck)

<p align="right">(<a href="#top">back to top</a>)</p>



<!-- ACKNOWLEDGMENTS -->
## Acknowledgments

I've used the following projects as dependencies.

* [sijms/go-ora](https://github.com/sijms/go-ora)
* [schollz/progressbar](https://github.com/schollz/progressbar)
* [olekukonko/tablewriter](https://github.com/olekukonko/tablewriter)

<p align="right">(<a href="#top">back to top</a>)</p>

<!-- MARKDOWN LINKS & IMAGES -->
<!-- https://www.markdownguide.org/basic-syntax/#reference-style-links -->
[contributors-shield]: https://img.shields.io/github/contributors/sixcolors/orasuck.svg?style=for-the-badge
[contributors-url]: https://github.com/sixcolors/orasuck/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/sixcolors/orasuck.svg?style=for-the-badge
[forks-url]: https://github.com/sixcolors/orasuck/network/members
[stars-shield]: https://img.shields.io/github/stars/sixcolors/orasuck.svg?style=for-the-badge
[stars-url]: https://github.com/sixcolors/orasuck/stargazers
[issues-shield]: https://img.shields.io/github/issues/sixcolors/orasuck.svg?style=for-the-badge
[issues-url]: https://github.com/sixcolors/orasuck/issues
[license-shield]: https://img.shields.io/github/license/sixcolors/orasuck.svg?style=for-the-badge
[license-url]: https://github.com/sixcolors/orasuck/blob/master/LICENSE.txt
[product-screenshot]: images/screenshot.png
