**(C) 2020 Geotab Inc**<br>

All files and artifacts in this repository are licensed under the provisions of the license provided by the LICENSE file in this repository.

# VISS v2 service manager

At statup the VISSv2 service manager tries to read a DB file named statestorage.db.
If successful, then the signal value(s) being requested will first be searched for in this DB, if not found then dummy values will be returned instead. 
Dummy values are always an integer, taken from a counter that is incremented every 37 msec, and wrapping to stay within the values 0 to 999.

If a request contains an array of paths, then the response/notification will include values related to all elements of the array. 

The service manager will do its best to interpret subscription filter expressions, but if unsuccessful it will return an error response without activating a subscription session.
