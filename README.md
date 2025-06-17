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
- **`[PUB]` POST** -> `/sign-up/send-code` - *send confirmation code*
- **`[PUB]` POST** -> `/sign-up/resend-code` - *resend confirmation code*
- **`[PUB]` POST** -> `/sign-up/verify` - *verify confirmation code and register*
- **`[PUB]` POST** -> `/sign-in/send-code` - *send 2fa code*
- **`[PUB]` POST** -> `/sign-in/resend-code` - *resend 2fa code*
- **`[PUB]` POST** -> `/sign-in/verify` - *verify 2fa code and log in*
- **`[PUB]` POST** -> `/refresh` - *refresh token pair*
- **`[AUTH]` PATCH** -> `/update-pw` - *update password*
- **`[PUB]` POST** -> `/request-fp-code` - *request forgot-password code to change password*
- **`[PUB]` PATCH** -> `/change-forgotten-pw-by-code` - *change forgotten password by requested code*

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
    - **PUT** -> `/update/socialLinks` - *add social link*
    - **DELETE** -> `/update/socialLinks` - *delete social link*
