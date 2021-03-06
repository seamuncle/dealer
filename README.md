The struggle between the things you know become really insane on a feed 
and the simple data in this example come to a head here.  When in doubt I leaned to simple,
but I'm discovering its hard to pretend there's things I don't know, for example, missing 
vehicles aren't tombstoned or marked sold, they're just deleted.

Did find a balance between the two, and ultimately got something both over- and under-built.
That said, it compiles and builds and does everything I wanted it to.

Given a lot more time, most of the code is written to be testable, and tests could be added.  
But, in the end, I spent a lot of time just debugging and logging and nothing obvious jumped out 
at me--but I can't memeber the last time I made this many assumptions about anything.

Experience-wise, I haven't touched an ORM or sqllite before--in the end, gorm seemed to cover both pretty well, 
and  proved remarkably more stable than I could have hoped--I did run through a couple libraries that were less helpful
 till I landed on this.  The gist of an ORM here is it lets us use go tags to map data more simply than manually
 generating whatever db dialects are at play

Given a go 1.13 environment set up somewhere UNIXy, running this against the included db file
*should be* a simple matter of:

```shell
$ go get -v github.com/seamuncle/dealer
$ cd $GOROOT/src/github.com/seamuncle/dealer
$ go get ./...
$ go build ./cmd/import 
$ $GOROOT/bin/import
```

Reminder, it will update the file. and I left the orm debugging on, so you can see how it does this.

The `main.go` file lives at github.com/seamuncle/dealer/cmd/import/main.go

Like so many other Go things, the layout bay not be intuitive, it reflects go's need for non circular import dependencies
--where a leaf package living in a higher directory, may include packages with, that aren't something it requires, 
but rather things that require it in its directory leaves.  Packages with no shared concerns are likely to reside further
 outseide of a directory tree from one another--but putting everyting in a wide structure can be tedious and makes it harder to follow where known dependencies lie and let cycles creep inadvertently in. For example given 2 github projects, you can't quickly tell what their dependency relationship is.

As everything requires the model, it lives in the root package, I called `dealer` --files within a package implicitly are allowed to have unprotected access to each other--there's differnt rules about concerns you might use to split them the files--none of them great, but "by business concern" is one of the better ones imho i.e. you can and should mix your models, interfaces, controllers, etc in a single file so long as they are strongly relevent to one-anothers concerns.

I created a sub package called `importer`, that does high-level business logic for a simple import and it 
resides under `dealer` as it explicitly requires the `dealer` package.  

Individual versions of the `main` package (its not uncommon to see a project ship with more than 1 binary) usually reside
in parallel under `cmd` in folder named for the binary--in this case, a binary called `import` so the actual folder 
is `cmd/import` which has a package called `main`, not `import` (which is a keyword and would lead to madness)

I think this covers the big things.