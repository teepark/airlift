/*
Airlift is a simple program that transfers anything over a local network.

Feed it data on stdin to publish, otherwise it will look for an existing
airlift and dump it to stdout.

Transfer an image to a friend:

    $ airlift <funny.jpg

and she can receive it with:

    $ airlift >funny.jpg
*/
package main
