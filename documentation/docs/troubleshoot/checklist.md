# Troubleshooting checklist

The following questions might help you identify the origin of the problem while using Postgres1st:
{.power-number}

1. Are you using the latest PGF WatchTower version?
2. Did you check the known issues section in the Release Notes for that particular PGF WatchTower release?
3. Are you receiving any error messages?
4. Do the logs contain any messages about the problem? See Message logs and Trace logs for more information.
5. Does the problem occur while configuring PGF WatchTower, such as:
     - Does the problem occur while you configure a specific function?
     - Does the problem occur when you perform a particular task?
6. Are you using the recommended [authentication](../api/authentication.md#authenticate) method?
7. Does your system’s firewall allow TCP traffic on the [ports](../install-pmm/plan-pmm-installation/network_and_firewall.md#essential-ports) used by PGF WatchTower?
8. Have you allocated enough disk space for installing PGF WatchTower? If not, check the disk allocation space.
9. Are you using a Technical Preview feature? Technical Preview features are not production-ready and should only be used in testing environments. For more information, see the relevant Release Notes.
10. For installing the PGF WatchTower client, are you using a package other than a binary package without root permissions?
11. Is your [PGF WatchTower Server](../install-pmm/install-pmm-server/index.md) installed and running with a known IP address accessible from the client node?
12. Is the [PGF WatchTower Agent](../install-pmm/install-pmm-client/index.md) installed, and is the node registered with PGF WatchTower Server?
13. Is PGF WatchTower Agent configured correctly and has access to the config file?
14. For monitoring MongoDB, do you have adminUserAnyDatabase or superuser role privilege to any database servers you want to monitor?
15. For monitoring Amazon RDS using PGF WatchTower, is there too much latency between PGF WatchTower Server and the Amazon RDS instance?
16. Have you upgraded the PGF WatchTower Server before you upgraded the PGF WatchTower Agent? If yes, there might be configuration issues, thus leading to failure in the client-server communication, as PGF WatchTower Server might not be able to identify all the parameters in the configuration.
17. Is the PGF WatchTower Server version higher than or equal to the PGF WatchTower Agent version? Otherwise, there might be configuration issues, thus leading to failure in the client-server communication, as PGF WatchTower Server might not be able to identify all the parameters in the configuration.


