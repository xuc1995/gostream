# gostream
A go module supplying Java-Like generic stream programming (while do type check at runtime)

## Using
### Get a Stream
To get a Stream, using SliceStream(yourSlice).
To get a EntryStream in which map key-value entry, useing EntryStream(youMap)
Because at that time point, the doc is very limited, check *_test.go to figure out how to use it

## Warning
At that point, runtime type check haven't been implemented, as to say, it will PANIC if you pass a parameter with a wrong type (as it is held as interface{}, it will not be checked and compile time).

I am going to add runtime type check next step, so it may return error instead of just PANIC in the nearly future.
