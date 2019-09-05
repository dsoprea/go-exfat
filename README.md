# Overview

An exFAT implementation based on the Microsoft-published specs:

https://docs.microsoft.com/en-us/windows/win32/fileio/exfat-specification


# Goals

To create a reader-only implementation of exFAT that enabled access using
familiar and accepted Go patterns.


# Status

Complete.


# Notes

- All entry-types are parsed as per the requirements of the specification.
  However:

  - Up-case tables, which support case insensitivity, are not read and therefore
    not applied. As a result, all file-operations are case-sensitive (and the
    villagers rejoiced).

  - Allocation bitmaps are not read, so it's not possible to know which clusters
    are or are not used. This is not requird for browsing the filesystem or
    reading files.

- Timestamps are accurate to one second.
