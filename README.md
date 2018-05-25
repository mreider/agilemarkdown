# agilemarkdown
A framework for managing backlogs using git and markdown

## Documentation

Read the docs at [http://agilemarkdown.com](http://agilemarkdown.com)

## Installing the agilemarkdown CLI

Make sure you have [GO](https://golang.org/doc/install) installed


Get the Go library


```
go get -u github.com/mreider/agilemarkdown
```

Compile the code


```
cd $GOPATH/src/github.com/mreider/agilemarkdown
./build.sh
```

Create an alias for the binary


```
$GOPATH/bin/agilemarkdown alias am
```

## Serving agilemarkdown projects from a web server

Since AgileMarkdown uses markdown and git - you can serve the files a few different ways. We prefer Caddy Server.

1. [Caddy server](https://caddyserver.com) - with markdown enabled
2. Use github repositories or github wiki
3. [Gollum](https://github.com/gollum/gollum/) - the project beneath github wiki
4. [Realms](https://github.com/scragg0x/realms-wiki) - similar to Gollum but in Python

## Sample Caddy Server setup

[Caddy server](https://caddyserver.com) is easy to set up. 

### Sample Caddy File

This caddy file would serve your AgileMarkdown project from the example.org/project folder. It is also configured to pull in some templates user files which are shown below. Note that this caddy configuration supports Google oAuth. You can read about other options at [https://caddyserver.com](https://caddyserver.com).

```
localhost:8081, example.org {
 gzip
 log example.org /access.log
 root example.org 

 markdown / {
 ext .md
 template template.html
 }

 jwt {
    path /
    redirect /login
    allow role user
    except /css
 }

 login {
    success_url /project
    google cddd,client_secret=dddd,scope=https://www.googleapis.com/auth/userinfo.email
    user_file example.org/users.yml
    template login.html
 }

}
```

### Sample user file

```
- domain: example.org
  origin: google
  claims:
    role: user

- claims:
  role: unknown
```

### Sample template file

```
<!DOCTYPE html>
<html lang="en">
<head>
  <title>Agilemarkdown</title>
  <meta charset="utf-8">
  <link rel="stylesheet" href="https://example.org/css/styles.css">
</head>
<body body style="margin:20px;padding:20px">
<br>
<div class="container">
  {{.Doc.body}}
</div>

</body>
</html>
```

### Example CSS

We love the [Markdown CSS](https://css-pkg.github.io/style.css/) provided by [Nate Goldman](https://github.com/ungoldman) and [Bret Comnes](https://github.com/bcomnes)

### Example login template

```
<!DOCTYPE html>
<html lang="en">
<head>
  <title>Example</title>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" href="https://example.org/css/styles.css">
</head>
<body>
<br>
<div class="container-fluid">
      {{ if .Error}}
        <div class="alert alert-danger" role="alert">
          <strong>Internal Error. </strong> Please try again later.
        </div>
      {{end}}

      Note: if you keep getting redirected to this page it means
      your google account is not authorized to visit this site.

      {{if .Authenticated}}

         {{template "userInfo" . }}

      {{else}}

        {{template "login" . }}

      {{end}}
</div>
</body>
</html>
```

## Providing a markdown editor in the browser

You can provide a markdown editor in the browser by using [pendulum](https://github.com/titpetric/pendulum). Simply download it and put it somewhere in your path.

Next, you can start pendulum and pipe output to dev/null to keep using the command line:

`pendulum -port 8004 -contents example.org/project/ &>/dev/null &`

In your caddy file, you can serve pendulum via a different port. Your users can navigate to edit files, and move around to different directories, if they want to.

At the bottom of your caddyfile:

```
example.org:666 {
gzip
log example.org/access.log
root example.org

proxy / localhost:8004
jwt {
   path /
   redirect /login
   allow role user
   except /css
}

login {
   success_url /project
   google client_id=fff,client_secret=fff,scope=https://www.googleapis.com/auth/userinfo.email
   user_file example.org/users.yml
   template login.html
}


}
```

Here's what it looks like:

![Pendulum Editor](https://monosnap.com/image/hk2qbU5nXlaXMQRA5BNTae1HgYfcj6.png)
