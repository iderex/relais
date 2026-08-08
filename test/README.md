# test

Suites that judge the repository rather than one package of it. What belongs
here is a test whose subject is a property of the tree as a whole, so that it has
no package to sit inside without picking one arbitrarily and making that package
look like its owner.

`architecture/` holds the dependency directions from
[the layout record](../docs/architecture.md) as a table a machine refuses a
violation of. It reads the tree as files rather than as packages, so a direction
that would not compile is still measurable, and it imports nothing else in this
repository, so it cannot become a route around the boundaries it judges.

What does not belong here is a test with an obvious package. A unit test lives
beside the code it is about, where somebody changing that code will see it.
