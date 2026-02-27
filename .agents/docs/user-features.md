# owui users command

owui users command is the subcommand wich manages users in open webui. Refer and search through [Open WebUi v0.8.5 OpenAPI](file:///Users/christoph/projekte/owui-go/openapi-reference/0.8.5-openapi.json).

## List users
```bash
owui users list|ls
```
This command lists all users in open webui with their id, username, email and role.
```example terminal output
$ owui users list
ID    USERNAME    EMAIL               ROLE
1     alice       alice@example.com   admin
2     bob         bob@example.com     user
3     charlie     charlie@example.com pending
```

## Create user
```bash
owui users create --username <username> --email <email> --password <password> --role <role>
```
This command creates a new user in open webui with the specified username, email, password and role. The role can be either `admin` or `user` and `pending`. when users create without the options an interactive prompt will be shown to input the required information like we do in `owui instaces add`

```example terminal output
$ owui users create
? Username: alice
? Email:
? Password: ********
? Role: (Use arrow keys)
    admin
    user
    pending
Successfully created user alice with role admin
```

## Delete user
```bash
owui users remove|rm <user_name>
```
```
This command deletes a user from open webui by their username. You will be prompted to confirm the deletion before it is executed. we will ask for confirmation before deleting the user to prevent accidental deletion. Also we need auto completion for the username to make it easier to select the user to delete.
If no username is provided the command will show an interactive prompt to select a user to delete. The prompt will also have a search functionality to filter the users by their username. This will make it easier to select the user to delete, especially when there are many users in open webui.

```shell example terminal output
$ owui users remove
? Select user to delete (Use arrow keys)
search: ali
    alice
    charlie
Confirm deleting user alice? (y/n) y
Successfully deleted user alice
```

## Update users role
```bash
owui users update-role <user_name> --role <user|admin|pending>
```
This command updates the role of a user in open webui by their username. The new role can be either `admin` or `user` and `pending`. You will be prompted to confirm the update before it is executed. we will ask for confirmation before updating the user's role to prevent accidental changes. Also we need auto completion for the username to make it easier to select the user to update.
If no username is provided the command will show an interactive prompt to select a user to update. The prompt will also have a search functionality to filter the users by their username. This will make it easier to select the user to update, especially when there are many users in open webui.

```shell example terminal output
$ owui users update-role
? Select user to update role (Use arrow keys)
search: ali
    alice
    charlie
? Select new role: (Use arrow keys)
    admin
    user
    pending
Confirm updating user alice's role to admin? (y/n) y
Successfully updated user alice's role to admin
```

## Add user to group
```bash
owui users add-to-group <user_name> --group <group_name>
```
- This command adds a user to a group in open webui by their username and the group name. We need auto completions for both the username and the group name to make it easier to select the user and the group. 
- Only users with role user can be selected and added to groups. the auto completion need to filter out users with admin role and pending role to prevent adding them to groups.
- A user can only be added to "local" groups wich means that groups with the description "Group <group_name> created automatically via OAuth." need to be filtered out from the auto completion list. You will be prompted to confirm the addition before it is executed.
- The command can also add multiple users to a group by providing multiple usernames separated by space. For 

example:

```bash
owui users add-to-group user1 user2 user3 --group group1
```

when no user is provided the command will show an interactive prompt to select multiple users to add to the group as checkbox style. The prompt will also have a search functionality to filter the users by their username. This will make it easier to select the users to add to the group, especially when there are many users in open webui.

```shell example terminal output
$ owui users add-to-group

? Select users to add to group (Use arrow keys, space to select)
search: bo
    [ ] alice
    [x] bob
    [ ] charlie
    [x] diana

? Select group (Use arrow keys)
search: dev
    developers
    engineers
    designers

Confirm adding 2 users to group 'developers'? (y/n) y
Successfully added bob, diana to group developers
```

## Delete user from group

```bash
owui users remove-from-group <user_name> --group <group_name>
```

- This command removes a user or multiple users from a group in open webui by their username and the group name. We need auto completions for both the username and the group name to make it easier to select the user and the group. 
- We only can remove users from "local" groups wich means that groups with the description "Group <group_name> created automatically via OAuth." need to be filtered out from the auto completion list. You will be prompted to confirm the removal before it is executed.
- The command can also remove multiple users from a group by providing multiple usernames separated by space. For example:

```bash
owui users remove-from-group user1 user2 user3 --group group1
```

when no user is provided the command will show an interactive prompt to select multiple users to remove from the group as checkbox style. The prompt will also have a search functionality to filter the users by their username. This will make it easier to select the users to remove from the group, especially when there are many users in open webui.

```shell example terminal output
$ owui users remove-from-group
? Select users to remove from group (Use arrow keys, space to select)
search: bo
    [ ] alice
    [x] bob
    [ ] charlie
    [x] diana

? Select group (Use arrow keys)
search: dev
    developers
    engineers
    designers
Confirm removing 2 users from group 'developers'? (y/n) y
Successfully removed bob, diana from group developers
```



if no instance is specified with `--instance` or `-i` flag, the command will use the active instance. You can switch the active instance with `owui instances use <instance_name>`.