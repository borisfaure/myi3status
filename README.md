A wrapper on i3status written in go


Main Feature
============

Display whether it'll rain in the next hour in one of the supported places in
France.

i3 config usage example
=======================
```
bar {
	#status_command ~/.local/bin/myi3status "-location=lat=48.892576&lon=2.287438"  #Courbevoie
	status_command ~/.local/bin/myi3status "-location=lat=45.758097&lon=4.8407"  #Lyon
	tray_output primary
	position top
}
```

How to find the latitude and longitude arguments
================================================

- Open up meteo france's website at your chosen city, for example https://meteofrance.com/previsions-meteo-france/lyon/69000
- Open up the developer tools in your web browser and go to the network tab
- Click the "Mettre à jour" button in the "Pluie dans l'heure" section
- Copy the latitude and longitude parameters from the request that appeared in the network tab of your web brower's developer tools.
