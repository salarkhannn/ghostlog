#!/bin/sh
set -e

# Detect OS and Architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if [ "$ARCH" = "x86_64" ]; then
    ARCH="x86_64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    ARCH="arm64"
elif [ "$ARCH" = "i386" ] || [ "$ARCH" = "i686" ]; then
    ARCH="i386"
else
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

if [ "$OS" = "darwin" ]; then
    OS="Darwin"
elif [ "$OS" = "linux" ]; then
    OS="Linux"
fi

REPO="salarkhannn/ghostlog"
echo "Fetching latest release for $OS ($ARCH)..."

# Use GitHub API to find the latest release download URL
LATEST_URL=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" \
    | grep '"browser_download_url":' \
    | grep "${OS}_${ARCH}\.tar\.gz" \
    | cut -d '"' -f 4 \
    | head -n 1)

if [ -z "$LATEST_URL" ]; then
    echo "Error: Could not find a precompiled binary for ${OS}_${ARCH}."
    echo "Please check the releases page manually: https://github.com/$REPO/releases"
    exit 1
fi

echo "Downloading from $LATEST_URL..."
curl -sL "$LATEST_URL" | tar xz -C /tmp ghostlog

INSTALL_DIR="/usr/local/bin"

echo "Installing to $INSTALL_DIR (may require sudo password)..."
if [ -w "$INSTALL_DIR" ]; then
    mv /tmp/ghostlog "$INSTALL_DIR/ghostlog"
else
    sudo mv /tmp/ghostlog "$INSTALL_DIR/ghostlog"
fi

chmod +x "$INSTALL_DIR/ghostlog"

cat << 'EOF'
                     XX$X$$$$&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&                          
                  &X$&&&&&$+:                          +                & &  &                    
               X&&&&&&X:   :. ...:   :.: ++.+...  : & & X&   : + &+&&X& & x .   &                 
             X$$&&&&+:::.:.:+++:+:.:.....:.:  .....    :X.  X       x + .::&&     &               
            X$&&&&x+::+++::+::+::::.:.: .. ::::.x ..x   $ . +   ++  ++ .:   &   .    &            
           X$&&&X:.:++x+:+x:++::....::.+.:  .. . .. :.    .:.::      X & X    &  :+   &           
          &&&&&::x:.    .: :..::::: ..:.:::... .. ..:.:..:...   . .                               
          &&&+:....         .::+:.+.+.. :  : ::: :.   ..  . ::.:...   &+x :  x.  .    &           
         &&&.+:.+:: &&&&     .  ... :.:.: ...      . .  :  .     ..   :x &&+ X   +:.+             
          && +x+++:  &&&&&     .+.:.: . ....  ::..... ..  .      .. .                  &&         
          &:  .:++.    &&&&&      ..: +. . ..  . . ...   .  .. :.   . . .   .          &          
          & ::+:.::+    &&&&&.      . .: .:.. .. .     . .  .         .   . X  . . .   &          
          &   :.:+ .      &&&&&     .  .  . .    .  .     .              .   : . .     &          
          & .:+:+: ...      &&&&&   . ....        .  .                      ..         &          
          &   ..:.::.+..     &&&&&   ... : .                                           &          
          &  :+... ..      &&&&&     . .   .  .             &&&&&&&&&&&&&&        . .  &          
          &  .: ::.+      &&&&         .         .       &&&&&&&&&&&&&&&& $&&          &          
          &  .: .:      &&&&&               .  .      .&&&&&&&&&&&&:     :x. X&        &          
          &   :..     &&&&&                          &&&&&&&&&&&x.:+X:+:+: &: &&&      &          
          & ..  ..  &&&&&     .  &&&&&&&&&&+ .     .&&&&&&&&&&+:xx+++.  :        &     &          
          &  .:.:.   &&       .  &&&&&&&&&&&       &&&&&&&&X::+X++++.++. ..     . &    &          
          & :....         .                       &&&&&&$&&&&&X :+:+.            & &   &          
          &   .  . .      . .  .                 &X&&&&x$&    &X ++: . &$xX  .     &   &          
          &    ..      . ..            :     .   & &&Xx+&+     +:+   &     X      &&   &          
          &  & .   : X .         .  .       .   &X&&$xxx&     &x.++ $  :+         &:   &          
          &               :  :.          .      & & x+x+&     &  :  &     &       &$   &          
          &   & $  : &      .   . ..  .     .   & &:++++&     &   . &  :          &&   &          
          &&&&  &x  &&  .    .           .      :& .:.x &&&&&&X  .: $    &   .   + &   &          
          &  X&                .. . .  .       & &.x&:+:.      :... :x&&.    : .:& &   &          
          &     ..:  :  :  .        .  .       X & .:+x:+.::.:  ..: .      : .   & &   &          
          &              .        . .  .      & &+:     . : .: .. .  . .     : . & &   &          
          & &&+ &x X&: + . .  ..             & & X $&$ .:.  : . . . ..:..:.. .   &&&   &          
          &                     .           &:& X&     .  .:  :: ..  .   .        & &  &          
          &                .     .         &:& X   . .  ... ... ...          ..   $&&  &&         
          & +&& +X   & & :  . . :       &&&& +   $   .$ :.  . . .  .+ : &$$.x :.   x &            
             x& $$ &X&  : x   +. .    &&&&&:&: :   .+ . ...: ...: :  .x X & &  :  &&&&&           
           &                .        &&&.          X  :. ......   .                 x&&&          
            &   +. ++:& .  : :.. :   &         .      x  :  .. . .   &X  &&x&& && &   &&&         
               +$& + . ++x   . . .:  &       X &X&.      $ :   + +.  $ :  &     x && & + $        
                             .+       &&&&&&&&    &x&        ::    X    & ::&&&        . $        
                 +          .  . : X                &     :.    .  :    XX&&& &&&&&     &         
                   + .x +                  +     ..  &:X    x     & & x&&&        &&x:&&          
                    &      +:    . .  +       +       X&.&         &&&&&                          
                                                         &   &&&&&.+ &                            
EOF

echo ""
echo "=========================================="
echo "✨ ghostlog installed successfully!"
echo "=========================================="
echo "Run 'ghostlog' in your terminal to start."
