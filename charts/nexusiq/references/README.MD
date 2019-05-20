### Reference Documentation for Nexus IQ Server config.yml

The IQ Server is an application exposed using a Dropwizard server.
The main configuration file for the IQ Server installation is a YAML formatted file called config.yml found in the installation directory. The config.yml file typically contains only those configuration options which are rarely changed.
Special considerations when editing the config.yml file:
TAB characters are not supported, use space characters only for indenting
structure is tree-like - indents define structure hierarchy and are relevant to proper parsing of the file
indented lines are considered child options of the first un-commented outdented line preceeding them
commented lines are ignored - comments begin with the # character
an improperly formatted config.yml will prevent the server from starting
We strongly recommend using a text editor that will inform you of any TAB characters accidentally inserted into the file.
Options that are more commonly changed are typically found in the System Preferences section of the IQ Server user interface, which you can access by clicking on the System Preferences icon located in the top right of the IQ Server header ().

## Useful Links:
1. Documentation on config.yml - https://help.sonatype.com/iqserver/configuring/config.yml 

The config.yml file is stored by default in the container in the following directory: ~/opt/sonatype//opt/sonatype/nexus-iq-server/config.yml
This 