package aero

import (
	"fmt"

	"github.com/Alsond5/aero/internal/color"
)

const banner = `
      __       _______   _______     ______    
     /""\     /"     "| /"      \   /    " \   
    /    \   (: ______)|:        | // ____  \  
   /' /\  \   \/    |  |_____/   )/  /    ) :) 
  //  __'  \  // ___)_  //      /(: (____/ //  
 /   /  \\  \(:      "||:  __   \ \        /   
(___/    \___)\_______)|__|  \___) \"_____/    
                                               
`

func printBanner(addr string) {
	fmt.Println(color.Colorize(color.Purple, banner))
	fmt.Printf("%s  => http server started on %s\n",
		color.BrightBlack,
		color.Colorize(color.Green, addr)+color.Reset,
	)
}
