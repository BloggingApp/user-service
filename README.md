# user-service

#### Please note that the main code under development is in the [dev](https://github.com/BloggingApp/user-service/tree/dev) branch

### API Docs

`/api/v1` - base route

**Headers**:
- **`Authorization`**: Bearer `<ACCESS_TOKEN>`

**Designations**:
- **`[AUTH]`** - ***requires** auth*
- **`[PUB]`** - ***doesn't** require auth*

`/auth`:
- **`[PUB]` POST** -> `/sign-up/send-code` - *send registration code*
- **`[PUB]` POST** -> `/sign-up/verify` - *verify registration code and register*
- **`[PUB]` POST** -> `/sign-in/send-code` - *send confirmation code for log in*
- **`[PUB]` POST** -> `/sign-in/verify` - *verify **two-factor authentication** code and log in*
- **`[PUB]` POST** -> `/refresh` - *refresh token pair*

---

`/users`:
- **`[AUTH]` GET** -> `/byUsername/:<username>` - *get user by username*
- **`[AUTH]` PUT** -> `/follow/:<userID>` - *follow user*
- **`[AUTH]` DELETE** -> `/unfollow/:<userID>` - *unfollow user*
- **`[AUTH]` PATCH** -> `/:<userID>/notifications` - *enable/disable notifications about new **:userID**'s posts*

- **`[AUTH]`** `/@me`:
    - **GET** -> `/` - *get authorized user info*
    - **GET** -> `/followers` - *get user followers*
    - **GET** -> `/follows` - *get user followed channel*
    - **PATCH** -> `/update` - *update user info*
    - **PATCH** -> `/update/setAvatar` - *set avatar*
    - **PATCH** -> `/update/updatePassword` - *update password*
    - **PUT** -> `/update/socialLinks` - *add social link*
    - **DELETE** -> `/update/socialLinks` - *delete social link*
